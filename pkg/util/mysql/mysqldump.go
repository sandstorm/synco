package mysql

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/sandstorm/synco/pkg/common"
	mysqldump "github.com/sandstorm/synco/pkg/util/mysql/go_mysqldump"
	"io"
)

func CreateDump(dbCredentials *common.DbCredentials, writer io.WriteCloser, whereClauseForTables map[string]string) (*sql.DB, error) {
	// Open connection to database
	config := mysql.NewConfig()
	config.User = dbCredentials.User
	config.Passwd = dbCredentials.Password
	config.DBName = dbCredentials.DbName
	config.Net = "tcp"
	config.Addr = fmt.Sprintf("%s:%d", dbCredentials.Host, dbCredentials.Port)

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	// Register database with mysqldump

	dumper := mysqldump.NewDumper(db, writer)
	dumper.WhereClauseForTables = whereClauseForTables
	err = dumper.Dump()
	if err != nil {
		return nil, fmt.Errorf("error registering database: %w", err)
	}

	// Close dumper, connected database and file stream.
	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("error closing dumper: %w", err)
	}

	return db, nil
}
