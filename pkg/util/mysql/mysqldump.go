package mysql

import (
	"database/sql"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"github.com/jamf/go-mysqldump"
	"github.com/sandstorm/synco/pkg/frameworks"
)

func CreateDump(dbCredentials *frameworks.DbCredentials, folderName string, filenamePrefix string) (resultFile string, err error) {
	// Open connection to database
	config := mysql.NewConfig()
	config.User = dbCredentials.User
	config.Passwd = dbCredentials.Password
	config.DBName = dbCredentials.DbName
	config.Net = "tcp"
	config.Addr = fmt.Sprintf("%s:%d", dbCredentials.Host, dbCredentials.Port)

	db, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		return "", fmt.Errorf("error opening database: %w", err)
	}

	// Register database with mysqldump
	dumper, err := mysqldump.Register(db, folderName, filenamePrefix)
	if err != nil {
		return "", fmt.Errorf("error registering database: %w", err)
	}

	// Dump database to file
	err = dumper.Dump()
	if err != nil {
		return "", fmt.Errorf("error dumping: %w", err)
	}

	// Close dumper, connected database and file stream.
	err = dumper.Close()
	if err != nil {
		return "", fmt.Errorf("error closing dumper: %w", err)
	}
	return folderName + filenamePrefix + ".sql", nil
}
