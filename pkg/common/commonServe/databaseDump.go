package commonServe

import (
	"database/sql"
	"github.com/pterm/pterm"
	"github.com/sandstorm/synco/v2/pkg/common"
	"github.com/sandstorm/synco/v2/pkg/common/dto"
	"github.com/sandstorm/synco/v2/pkg/serve"
	"github.com/sandstorm/synco/v2/pkg/util/mysql"
)

func DatabaseDump(transferSession *serve.TransferSession, dbCredentials *common.DbCredentials, whereClauseForTables map[string]string) *sql.DB {
	// 2) DATABASE DUMP
	// basically the way it works is:
	// mysql.CreateDump --> age.Encrypt --> write to file.
	// but because this is based on streams, we need to construct it the other way around:
	// 1st: open the target file
	// 2nd: init age.Encrypt
	// 3rd: do mysql dump (which feeds the Writer)
	wc, err := transferSession.EncryptToFile("dump.sql.enc")
	fileSet := &dto.FileSet{
		Name: "dbDump",
		Type: dto.TYPE_MYSQLDUMP,
		MysqlDump: &dto.FileSetMysqlDump{
			FileName: "dump.sql.enc",
		},
	}

	// 2b) the actual DB dump. also finishes writing.
	db, err := mysql.CreateDump(dbCredentials, wc, whereClauseForTables)
	if err != nil {
		pterm.Fatal.Printfln("could not create SQL dump: %s", err)
	}
	fileSet.MysqlDump.SizeBytes = wc.Size()
	transferSession.Meta.FileSets = append(transferSession.Meta.FileSets, fileSet)
	err = transferSession.UpdateMetadata()
	if err != nil {
		pterm.Fatal.Printfln("could not update SQL dump metadata: %s", err)
	}

	pterm.Info.Printfln("Stored Database Dump in %s", "dump.sql.enc")
	return db
}
