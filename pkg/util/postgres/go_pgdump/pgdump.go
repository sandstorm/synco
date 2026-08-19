package go_pgdump

import (
	"context"
	"database/sql"
	"fmt"
	"io"
)

// headerSQL is written verbatim at the top of the dump so it replays
// cleanly via `psql -f`.
const headerSQL = `SET statement_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;
`

// Data configures and drives a Postgres dump, mirroring the shape of
// pkg/util/mysql/go_mysqldump.Data.
type Data struct {
	Out                  io.Writer
	Connection           *sql.DB
	WhereClauseForTables map[string]string
	IgnoreTables         []string

	tx *sql.Tx
}

// NewDumper registers a new dumper against an already-open connection.
func NewDumper(db *sql.DB, out io.Writer) *Data {
	return &Data{Connection: db, Out: out}
}

// Close closes the dumper's output (if closable) and its DB connection.
func (d *Data) Close() error {
	defer func() {
		d.Connection = nil
		d.Out = nil
	}()
	if out, ok := d.Out.(io.Closer); ok {
		_ = out.Close()
	}
	return d.Connection.Close()
}

func (d *Data) isIgnoredTable(name string) bool {
	for _, t := range d.IgnoreTables {
		if t == name {
			return true
		}
	}
	return false
}

func qualify(table string) string {
	return `"public".` + quoteIdent(table)
}

// qualifiedConstraint pairs a constraint with the (already schema-qualified)
// table it belongs to, so constraints from every table can be grouped and
// emitted together by type across the whole dump.
type qualifiedConstraint struct {
	qualifiedTable string
	c              constraint
}

// Dump writes a complete, replayable SQL dump of the database to d.Out.
//
// Phases, in order: (1) DROP + CREATE TABLE for every table (columns only -
// no constraints yet), (2) COPY data for every table, (3) PK, then UNIQUE
// constraints, then secondary indexes, then CHECK constraints, then FOREIGN
// KEY constraints last (deferred so referenced tables/rows already exist
// and no cross-table topological sort is needed), (4) sequence `setval`
// restoration. This mirrors real pg_dump's schema/data/post-data staging.
func (d *Data) Dump() error {
	tx, err := d.Connection.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
		ReadOnly:  true,
	})
	if err != nil {
		return err
	}
	defer tx.Rollback()
	d.tx = tx

	for _, stmt := range []string{
		"SET TIME ZONE 'UTC'",
		"SET extra_float_digits = 3",
		"SET bytea_output = 'hex'",
	} {
		if _, err := tx.Exec(stmt); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprint(d.Out, headerSQL); err != nil {
		return err
	}

	allTables, err := getTables(tx)
	if err != nil {
		return err
	}

	tables := make([]string, 0, len(allTables))
	for _, t := range allTables {
		if !d.isIgnoredTable(t) {
			tables = append(tables, t)
		}
	}

	copyColumnsByTable := make(map[string][]string, len(tables))

	// Phase 1: DROP + CREATE TABLE (columns only).
	for _, t := range tables {
		ddl, copyColumns, err := createTableDDL(tx, t)
		if err != nil {
			return err
		}
		copyColumnsByTable[t] = copyColumns

		if _, err := fmt.Fprintf(d.Out, "DROP TABLE IF EXISTS %s CASCADE;\n", qualify(t)); err != nil {
			return err
		}
		if _, err := fmt.Fprint(d.Out, ddl); err != nil {
			return err
		}
	}

	// Phase 2: data.
	for _, t := range tables {
		if err := streamCopy(d.Out, tx, t, copyColumnsByTable[t], d.WhereClauseForTables[t]); err != nil {
			return err
		}
	}

	// Phase 3: constraints + indexes, gathered across all tables first so
	// they can be emitted grouped by type (PK/UNIQUE/index/CHECK/FK) rather
	// than per table.
	var allConstraints []qualifiedConstraint
	var allIndexes []string
	for _, t := range tables {
		qualifiedTable := qualify(t)

		cs, err := getConstraints(tx, qualifiedTable)
		if err != nil {
			return err
		}
		for _, c := range orderConstraints(cs) {
			allConstraints = append(allConstraints, qualifiedConstraint{qualifiedTable, c})
		}

		idx, err := getIndexes(tx, qualifiedTable)
		if err != nil {
			return err
		}
		allIndexes = append(allIndexes, idx...)
	}

	writeConstraintsOfRank := func(rank int) error {
		for _, ac := range allConstraints {
			if constraintTypeRank[ac.c.conType] == rank {
				if _, err := fmt.Fprintln(d.Out, ac.c.alterStatement(ac.qualifiedTable)); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if err := writeConstraintsOfRank(constraintTypeRank["p"]); err != nil {
		return err
	}
	if err := writeConstraintsOfRank(constraintTypeRank["u"]); err != nil {
		return err
	}
	for _, idx := range allIndexes {
		if _, err := fmt.Fprintln(d.Out, idx); err != nil {
			return err
		}
	}
	if err := writeConstraintsOfRank(constraintTypeRank["c"]); err != nil {
		return err
	}
	if err := writeConstraintsOfRank(constraintTypeRank["f"]); err != nil {
		return err
	}

	// Phase 4: sequence state restoration.
	for _, t := range tables {
		stmts, err := sequenceRestoreStatements(tx, qualify(t))
		if err != nil {
			return err
		}
		for _, s := range stmts {
			if _, err := fmt.Fprintln(d.Out, s); err != nil {
				return err
			}
		}
	}

	return nil
}
