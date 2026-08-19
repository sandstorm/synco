package postgres

import (
	"database/sql"
	"fmt"
	"io"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sandstorm/synco/v2/pkg/common"
	go_pgdump "github.com/sandstorm/synco/v2/pkg/util/postgres/go_pgdump"
)

// CreateDump opens a connection to the given Postgres database and writes a
// complete, replayable SQL dump to writer.
//
// Unlike the MySQL dumper, there is no manual TLS-then-fallback retry here:
// `sslmode=prefer` negotiates TLS automatically and falls back to plaintext
// on its own if the server doesn't support it.
func CreateDump(dbCredentials *common.DbCredentials, writer io.WriteCloser, whereClauseForTables map[string]string) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=prefer",
		dbCredentials.Host, dbCredentials.Port, dbCredentials.User, dbCredentials.Password, dbCredentials.DbName,
	)

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %w", err)
	}

	dumper := go_pgdump.NewDumper(db, writer)
	dumper.WhereClauseForTables = whereClauseForTables
	if err := dumper.Dump(); err != nil {
		return nil, fmt.Errorf("error dumping database: %w", err)
	}

	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("error closing dumper: %w", err)
	}

	return db, nil
}
