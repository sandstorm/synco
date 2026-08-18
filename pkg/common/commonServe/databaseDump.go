package commonServe

import (
	"database/sql"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/v2/pkg/common"
	"github.com/sandstorm/synco/v2/pkg/common/dto"
	"github.com/sandstorm/synco/v2/pkg/serve"
	"github.com/sandstorm/synco/v2/pkg/util/mysql"
	"github.com/sandstorm/synco/v2/pkg/util/postgres"
)

// fileSetTypeFor maps a DB driver to the dto.FileSetType used to record its
// dump in the transfer session's metadata. Unrecognized/unset drivers
// default to mysql, matching the pre-Postgres-support behavior.
func fileSetTypeFor(driver common.DbDriver) dto.FileSetType {
	if driver == common.DbDriverPostgres {
		return dto.TYPE_POSTGRESDUMP
	}
	return dto.TYPE_MYSQLDUMP
}

func DatabaseDump(transferSession *serve.TransferSession, dbCredentials *common.DbCredentials, whereClauseForTables map[string]string) *sql.DB {
	// 2) DATABASE DUMP
	// basically the way it works is:
	// mysql.CreateDump / postgres.CreateDump --> age.Encrypt --> write to file.
	// but because this is based on streams, we need to construct it the other way around:
	// 1st: open the target file
	// 2nd: init age.Encrypt
	// 3rd: do the DB dump (which feeds the Writer)
	wc, err := transferSession.EncryptToFile("dump.sql.enc")
	if err != nil {
		pterm.Fatal.Printfln("could not open encrypted dump file: %s", err)
	}
	fileSet := &dto.FileSet{
		Name: "dbDump",
		Type: fileSetTypeFor(dbCredentials.Driver),
	}

	// 2b) the actual DB dump. also finishes writing.
	var db *sql.DB
	switch fileSet.Type {
	case dto.TYPE_POSTGRESDUMP:
		fileSet.PostgresDump = &dto.FileSetPostgresDump{FileName: "dump.sql.enc"}
		db, err = postgres.CreateDump(dbCredentials, wc, whereClauseForTables)
	default:
		fileSet.MysqlDump = &dto.FileSetMysqlDump{FileName: "dump.sql.enc"}
		db, err = mysql.CreateDump(dbCredentials, wc, whereClauseForTables)
	}
	if err != nil {
		pterm.Fatal.Printfln("could not create SQL dump: %s", err)
	}

	sizeBytes := wc.Size()
	if fileSet.PostgresDump != nil {
		fileSet.PostgresDump.SizeBytes = sizeBytes
	} else {
		fileSet.MysqlDump.SizeBytes = sizeBytes
	}
	transferSession.Meta.FileSets = append(transferSession.Meta.FileSets, fileSet)
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update SQL dump metadata: %s", err)
	}

	pterm.Info.Printfln("Stored Database Dump in %s", "dump.sql.enc")
	return db
}
