package main

import (
	"crypto/md5"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	"github.com/MouseHatGames/go-mysqldump"
	"github.com/MouseHatGames/go-mysqldump/internal/marshal"
)

func failIfErr(err error, msg string) {
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: %s\n", msg, err)
		os.Exit(1)
	}
}

var (
	tablesStr  = flag.String("tables", "", "comma-separated list of tables to export, if empty all tables will be exported")
	info       = flag.Bool("info", false, "only print information about the dump")
	verifyHash = flag.Bool("verify", false, "compare hash of the dump to a .md5 file")
)

func main() {
	flag.Parse()

	args := flag.Args()
	for _, v := range args {
		if len(args) > 1 {
			fmt.Printf("%s:\n", v)
		}

		err := doDump(v)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
		}

		fmt.Println()
	}
}

func doDump(dumpPath string) error {
	var in io.ReadSeeker
	if dumpPath == "" || dumpPath == "-" {
		in = os.Stdin
	} else {
		f, err := os.Open(dumpPath)
		if err != nil {
			return fmt.Errorf("failed to open dump file: %w", err)
		}
		defer f.Close()

		in = f
	}

	if *verifyHash && in != os.Stdin {
		v, err := verify(in, dumpPath)
		if err != nil {
			return fmt.Errorf("failed to verify dump hash: %w", err)
		}

		if v {
			fmt.Println("✔ Successfully verified dump")
		} else {
			fmt.Println("✖ Failed to verify dump integrity")
		}

		in.Seek(0, 0)
	}

	if *info {
		err := printInfo(in)
		if err != nil {
			return fmt.Errorf("failed to print dump info: %w", err)
		}
	}

	if *verifyHash || *info {
		return nil
	}

	var tables []string
	if *tablesStr != "" {
		tables = strings.Split(*tablesStr, ",")
	}

	err := mysqldump.ConvertToSQL(in, os.Stdout, mysqldump.ConvertOptions{
		Tables: tables,
	})
	if err != nil {
		return fmt.Errorf("failed to convert dump file: %w", err)
	}

	return nil
}

func verify(in io.Reader, dumpPath string) (bool, error) {
	hashPath := dumpPath + ".md5"

	b, err := ioutil.ReadFile(hashPath)
	if err != nil {
		return false, fmt.Errorf("read hash file at: %w", err)
	}

	hash := md5.New()

	_, err = io.Copy(hash, in)
	if err != nil {
		return false, err
	}

	hashStr := hex.EncodeToString(hash.Sum(nil))

	return hashStr == string(b), nil
}

func printInfo(in io.Reader) error {
	r := marshal.NewReader(in)

	fh, err := r.ReadFileHeader()
	if err != nil {
		return fmt.Errorf("read file header: %w", err)
	}

	fmt.Printf("Dump of database \"%s\" at %s\n", fh.DatabaseName, fh.DumpStart)
	fmt.Println("Server version", fh.ServerVersion)
	fmt.Println("Tables:")

	for {
		t, err := r.ReadTableHeader()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("read table header: %w", err)
		}

		fmt.Printf("   %s (%s)\n", t.Name, strings.Join(t.Columns, ", "))

		err = r.SkipRows(len(t.Columns))
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("skip data: %w", err)
		}
	}

	return nil
}
