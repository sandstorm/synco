package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/MouseHatGames/go-mysqldump"
	"github.com/MouseHatGames/go-mysqldump/internal/marshal"
)

func failIfErr(err error, msg string) {
	if err != nil {
		log.Prefix()
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	tablesStr := flag.String("tables", "", "comma-separated list of tables to export, if empty all tables will be exported")
	info := flag.Bool("info", false, "only print information about the dump")
	flag.Parse()

	dumpPath := flag.Arg(0)

	var in io.Reader
	if dumpPath == "" || dumpPath == "-" {
		in = os.Stdin
	} else {
		f, err := os.Open(dumpPath)
		failIfErr(err, "failed to open dump file")
		defer f.Close()

		in = f
	}

	if *info {
		failIfErr(printInfo(in), "failed to print dump info")
		return
	}

	var tables []string
	if *tablesStr != "" {
		tables = strings.Split(*tablesStr, ",")
	}

	err := mysqldump.ConvertToSQL(in, os.Stdout, mysqldump.ConvertOptions{
		Tables: tables,
	})
	failIfErr(err, "failed to convert dump file")
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
