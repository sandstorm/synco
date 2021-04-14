package main

import (
	"flag"
	"io"
	"log"
	"os"
	"strings"

	"github.com/MouseHatGames/go-mysqldump"
)

func failIfErr(err error, msg string) {
	if err != nil {
		log.Fatalf("%s: %s", msg, err)
	}
}

func main() {
	tablesStr := flag.String("tables", "", "")
	flag.Parse()

	dumpPath := flag.Arg(0)

	var in io.Reader
	if dumpPath == "" || dumpPath == "-" {
		in = os.Stdin
	} else {
		f, err := os.Open(dumpPath)
		failIfErr(err, "failed to open dump file")

		in = f
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
