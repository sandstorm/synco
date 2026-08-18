package go_pgdump

import (
	"database/sql"
	"fmt"
	"io"
	"strings"
)

// quoteIdent quotes a Postgres identifier (table/column/sequence name) for
// safe use in generated DDL, doubling any embedded double quotes.
func quoteIdent(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// copyEscape renders a single value in COPY text format: SQL NULL becomes the
// `\N` marker, everything else has backslash, tab, newline and carriage
// return escaped so the value can never be confused with the NULL marker or
// a row/column delimiter.
func copyEscape(v sql.NullString) string {
	if !v.Valid {
		return `\N`
	}

	s := v.String
	s = strings.ReplaceAll(s, `\`, `\\`)
	s = strings.ReplaceAll(s, "\t", `\t`)
	s = strings.ReplaceAll(s, "\n", `\n`)
	s = strings.ReplaceAll(s, "\r", `\r`)
	return s
}

// streamCopy writes a `COPY ... FROM stdin; ... \.` text-format block for
// the given table/columns to w. Every column is cast to ::text server-side
// so that bytea/json/array/uuid/timestamptz/numeric/boolean values all
// round-trip using only generic COPY-text escaping - no per-Postgres-type Go
// formatter is needed. An empty whereClause defaults to "TRUE" (dump
// everything), mirroring the MySQL dumper's WhereClauseForTables handling.
func streamCopy(w io.Writer, q queryer, table string, copyColumns []string, whereClause string) error {
	if whereClause == "" {
		whereClause = "TRUE"
	}

	qualifiedTable := `"public".` + quoteIdent(table)

	quotedCols := make([]string, len(copyColumns))
	castCols := make([]string, len(copyColumns))
	for i, col := range copyColumns {
		quotedCols[i] = quoteIdent(col)
		castCols[i] = quoteIdent(col) + "::text"
	}

	selectSQL := fmt.Sprintf("SELECT %s FROM %s WHERE %s", strings.Join(castCols, ", "), qualifiedTable, whereClause)
	rows, err := q.Query(selectSQL)
	if err != nil {
		return err
	}
	defer rows.Close()

	if _, err := fmt.Fprintf(w, "COPY %s (%s) FROM stdin;\n", qualifiedTable, strings.Join(quotedCols, ",")); err != nil {
		return err
	}

	values := make([]sql.NullString, len(copyColumns))
	dest := make([]interface{}, len(copyColumns))
	for i := range values {
		dest[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(dest...); err != nil {
			return err
		}
		escaped := make([]string, len(values))
		for i, v := range values {
			escaped[i] = copyEscape(v)
		}
		if _, err := fmt.Fprintf(w, "%s\n", strings.Join(escaped, "\t")); err != nil {
			return err
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}

	_, err = fmt.Fprint(w, "\\.\n")
	return err
}
