package go_pgdump

import (
	"bytes"
	"database/sql"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestQuoteIdent(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "plain name", in: "users", want: `"users"`},
		{name: "name containing a double quote", in: `fo"o`, want: `"fo""o"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := quoteIdent(tt.in)
			if got != tt.want {
				t.Errorf("quoteIdent(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestCopyEscape(t *testing.T) {
	tests := []struct {
		name string
		in   sql.NullString
		want string
	}{
		{name: "SQL NULL", in: sql.NullString{Valid: false}, want: `\N`},
		{name: "empty string is not NULL", in: sql.NullString{String: "", Valid: true}, want: ""},
		{name: "embedded tab", in: sql.NullString{String: "a\tb", Valid: true}, want: `a\tb`},
		{name: "embedded newline", in: sql.NullString{String: "a\nb", Valid: true}, want: `a\nb`},
		{name: "embedded carriage return", in: sql.NullString{String: "a\rb", Valid: true}, want: `a\rb`},
		{name: "embedded backslash", in: sql.NullString{String: `a\b`, Valid: true}, want: `a\\b`},
		{name: "literal backslash-N is escaped, not mistaken for NULL", in: sql.NullString{String: `\N`, Valid: true}, want: `\\N`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := copyEscape(tt.in)
			if got != tt.want {
				t.Errorf("copyEscape(%+v) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestStreamCopyEmitsExactBlockAndCastsColumnsToText(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"a", "b"}).
		AddRow("1", nil).
		AddRow("2", "has\ta\ttab")
	mock.ExpectQuery(`^SELECT "a"::text, "b"::text FROM "public"\."t" WHERE TRUE$`).WillReturnRows(rows)

	var buf bytes.Buffer
	err = streamCopy(&buf, db, "t", []string{"a", "b"}, "")
	assert.NoError(t, err)

	assert.Equal(t, "COPY \"public\".\"t\" (\"a\",\"b\") FROM stdin;\n"+
		"1\t\\N\n"+
		"2\thas\\ta\\ttab\n"+
		"\\.\n", buf.String())

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestStreamCopyUsesCustomWhereClause(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"a"})
	mock.ExpectQuery(`^SELECT "a"::text FROM "public"\."t" WHERE FALSE$`).WillReturnRows(rows)

	var buf bytes.Buffer
	err = streamCopy(&buf, db, "t", []string{"a"}, "FALSE")
	assert.NoError(t, err)

	assert.NoError(t, mock.ExpectationsWereMet())
}
