package mysql

import (
	"database/sql"
	"fmt"
	"io"

	"github.com/go-sql-driver/mysql"
	"github.com/sandstorm/synco/v2/pkg/common"
	mysqldump "github.com/sandstorm/synco/v2/pkg/util/mysql/go_mysqldump"
)

func CreateDump(dbCredentials *common.DbCredentials, writer io.WriteCloser, whereClauseForTables map[string]string) (*sql.DB, error) {
	// Open connection to database
	config := mysql.NewConfig()
	config.User = dbCredentials.User
	config.Passwd = dbCredentials.Password
	config.DBName = dbCredentials.DbName
	config.Net = "tcp"
	config.Addr = fmt.Sprintf("%s:%d", dbCredentials.Host, dbCredentials.Port)
	// Enable SSL usage but skip verification
	config.TLSConfig = "skip-verify"

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Register database with mysqldump

	dumper := mysqldump.NewDumper(db, writer)
	dumper.WhereClauseForTables = whereClauseForTables
	err = dumper.Dump()
	if err != nil {
		// NOTE: this case happens if TLS is not supported in a database -> we fallback to the other version without TLS
		return createDumpNoTls(dbCredentials, writer, whereClauseForTables)
	}

	// Close dumper, connected database and file stream.
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing dumper: %w", err)
	}

	return db, nil
}

func createDumpNoTls(dbCredentials *common.DbCredentials, writer io.WriteCloser, whereClauseForTables map[string]string) (*sql.DB, error) {
	// Open connection to database
	config := mysql.NewConfig()
	config.User = dbCredentials.User
	config.Passwd = dbCredentials.Password
	config.DBName = dbCredentials.DbName
	config.Net = "tcp"
	config.Addr = fmt.Sprintf("%s:%d", dbCredentials.Host, dbCredentials.Port)
	// DISABLE tls here

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Register database with mysqldump

	dumper := mysqldump.NewDumper(db, writer)
	dumper.WhereClauseForTables = whereClauseForTables
	err = dumper.Dump()
	if err != nil {
		return nil, fmt.Errorf("error registering database (with and without TLS): %w", err)
	}

	// Close dumper, connected database and file stream.
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing dumper: %w", err)
	}

	return db, nil
}
