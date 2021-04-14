package mysqldump

import (
	"errors"
	"fmt"
	"io"
	"log"
	"sort"

	"github.com/MouseHatGames/go-mysqldump/internal/marshal"
)

type ConvertOptions struct {
	// If nil, all tables will be converted. If a table is specified here but is not present on the dump, no error will be returned
	Tables []string
}

func ConvertToSQL(in io.Reader, w io.Writer, opts ...ConvertOptions) error {
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

	fmt.Fprintf(w, `-- Go SQL Dump %s
--
-- ------------------------------------------------------
-- Server version	%s

CREATE DATABASE IF NOT EXISTS %s;
USE %[3]s;

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;
 
`, version, h.ServerVersion, h.DatabaseName)

	for {
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
			r.SkipRows()
			continue
		}

		fmt.Fprintf(w, `--
-- Table structure for table %[1]s
--

DROP TABLE IF EXISTS %[1]s;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;

`, t.Name)

		w.Write([]byte(t.CreateSQL))

		fmt.Fprintf(w, `

/*!40101 SET character_set_client = @saved_cs_client */;
--
-- Dumping data for table %s
--
LOCK TABLES %[1]s WRITE;
/*!40000 ALTER TABLE %[1]s DISABLE KEYS */;

`, t.Name)

		rows, errs := r.ReadRows()

		if r, ok := <-rows; ok {
			fmt.Fprintf(w, "INSERT INTO %s VALUES ", t.Name)
			writeRow(w, r)
		}

	loop:
		for {
			select {
			case r, ok := <-rows:
				if !ok {
					break loop
				}
				w.Write(commaNewline)
				writeRow(w, r)

			case err, ok := <-errs:
				if ok && !errors.Is(err, io.EOF) {
					return err
				}
				break loop
			}
		}

		fmt.Fprintf(w, `;

/*!40000 ALTER TABLE %s ENABLE KEYS */;
UNLOCK TABLES;

-- Finished table data dump

`, t.Name)
	}

	return nil
}

func writeRow(w io.Writer, r marshal.RowData) {
	w.Write([]byte{'('})

	for i, v := range r {
		if v != nil {
			w.Write(quote)
			writeEscapedString(w, *v)
			w.Write(quote)
		} else {
			fmt.Fprint(w, "null")
		}

		if i < len(r)-1 {
			w.Write(comma)
		}
	}

	w.Write([]byte{')'})
}

// Taken from https://gist.github.com/siddontang/8875771
func writeEscapedString(w io.Writer, str string) {
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
			w.Write(b)
		} else {
			b[0] = c
			w.Write(b[0:1])
		}
	}
}
