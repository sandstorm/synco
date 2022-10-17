package mysqldump

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	"github.com/sandstorm/synco/pkg/util/mysql/go_mysqldump/internal/marshal"
)

type ConvertOptions struct {
	// If nil, all tables will be converted. If a table is specified here but is not present on the dump, no error will be returned
	Tables     []string
	SkipCreate bool
}

func ConvertToSQL(in io.Reader, w io.Writer, flusher chan<- bool, ready <-chan bool, querySize int, opts ...ConvertOptions) error {
	var opt ConvertOptions

	if len(opts) > 0 {
		opt = opts[0]
	}
	sort.Strings(opt.Tables)

	r := marshal.NewReader(in)

	h, err := r.ReadFileHeader()
	if err != nil {
		return fmt.Errorf("read file header: %w", err)
	}

	fmt.Fprintf(w, `-- Go SQL Dump %[1]s
--
-- ------------------------------------------------------
-- Server version	%[2]s
`, version, h.ServerVersion, "`"+h.DatabaseName+"`")

	if !opt.SkipCreate {
		fmt.Fprintf(w, `CREATE DATABASE IF NOT EXISTS %[3]s;
USE %[3]s;`, version, h.ServerVersion, "`"+h.DatabaseName+"`")
	}
	fmt.Fprintf(w, `/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
 
`)
	flusher <- false
	<-ready
	done := false
	for {
		if done {
			break
		}
		t, err := r.ReadTableHeader()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("read table header: %w", err)
		}

		// If the table is not in the options' table list, skip it
		i := sort.Search(len(opt.Tables), func(i int) bool {
			return opt.Tables[i] == t.Name
		})
		if len(opt.Tables) > 0 && i == len(opt.Tables) {
			log.Printf("skipping %s", t.Name)
			r.SkipRows(len(t.Columns))
			continue
		}

		if !opt.SkipCreate {
			fmt.Fprintf(w, `--
-- Table structure for table %[1]s
--

DROP TABLE IF EXISTS %[1]s;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;

`, t.Name)

			w.Write([]byte(t.CreateSQL))

			fmt.Fprint(w, `;

/*!40101 SET character_set_client = @saved_cs_client */;`)
		}
		fmt.Fprintf(w, `--
-- Dumping data for table %s
--


`, t.Name)
		flusher <- false
		<-ready

		rows, errs := r.ReadRows(len(t.Columns))

		truncated := false
		rowBytesWritten := 0
	loop:
		for {
			rowBytesWritten = 0
			if r, ok := <-rows; ok {
				if !truncated {
					fmt.Fprintf(w, `
						/*!40000 ALTER TABLE %[1]s DISABLE KEYS */;
						TRUNCATE %[1]s;`, t.Name)
					truncated = true
				}
				fmt.Fprintf(w, "REPLACE INTO %s(`%s`) VALUES ", t.Name, strings.Join(t.Columns, "`,`"))
				rowBytesWritten += writeRow(w, r)
			}

			for {
				select {
				case r, ok := <-rows:
					if !ok {
						break loop
					}
					w.Write(commaNewline)
					rowBytesWritten += writeRow(w, r)

					if rowBytesWritten > querySize {
						w.Write(semicolonNewline)
						flusher <- true
						<-ready

						continue loop
					}

				case err, ok := <-errs:
					if ok && !errors.Is(err, io.EOF) {
						return err
					}
					if errors.Is(err, io.EOF) {
						done = true
					} else if ok {
						break loop
					}
				}
			}
		}

		if rowBytesWritten > 0 {
			w.Write(semicolonNewline)
			flusher <- true
			<-ready
		}
		if truncated {
			fmt.Fprintf(w, `
/*!40000 ALTER TABLE %s ENABLE KEYS */;`, t.Name)
		}
		fmt.Fprint(w, `-- Finished table data dump`)
		flusher <- false
		<-ready

	}

	return nil
}

func writeRow(w io.Writer, r marshal.RowData) (l int) {
	w.Write([]byte{'('})
	l = 1

	for i, v := range r {
		if v != nil {
			w.Write(quote)
			l += 2
			l += writeEscapedString(w, *v)
			w.Write(quote)
			l += 2
		} else {
			fmt.Fprint(w, "null")
			l += 4
		}

		if i < len(r)-1 {
			w.Write(comma)
			l += 1
		}
	}

	w.Write([]byte{')'})
	l += 1
	return
}

// Taken from https://gist.github.com/siddontang/8875771
func writeEscapedString(w io.Writer, str string) (l int) {
	l = 0
	b := make([]byte, 2)

	var escape byte
	for i := 0; i < len(str); i++ {
		c := str[i]

		escape = 0

		switch c {
		case 0: /* Must be escaped for 'mysql' */
			escape = '0'
		case '\n': /* Must be escaped for logs */
			escape = 'n'
		case '\r':
			escape = 'r'
		case '\\':
			escape = '\\'
		case '\'':
			escape = '\''
		case '"': /* Better safe than sorry */
			escape = '"'
		case '\032': /* This gives problems on Win32 */
			escape = 'Z'
		}

		if escape != 0 {
			b[0] = '\\'
			b[1] = escape
			n, _ := w.Write(b)
			l += n
		} else {
			b[0] = c
			n, _ := w.Write(b[0:1])
			l += n
		}
	}
	return
}
