package go_pgdump

import (
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func TestGetTablesOnlyReturnsBaseTablesInPublicSchema(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows([]string{"relname"}).
		AddRow("t_child").
		AddRow("t_parent")

	// the query itself must filter to ordinary tables (relkind = 'r') in the
	// public schema, excluding views/matviews/partition children - a wrongly
	// shaped query (e.g. missing the relkind filter) will not match this
	// expectation and the test will fail.
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_class c.*relkind = 'r'.*NOT c\.relispartition`).
		WillReturnRows(rows)

	tables, err := getTables(db)
	assert.NoError(t, err)
	assert.Equal(t, []string{"t_child", "t_parent"}, tables)

	assert.NoError(t, mock.ExpectationsWereMet())
}

var columnQueryCols = []string{"attname", "attnum", "data_type", "attnotnull", "default_expr", "attidentity", "attgenerated"}

func expectColumnQuery(mock sqlmock.Sqlmock, rows *sqlmock.Rows) {
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_attribute a.*attisdropped`).WillReturnRows(rows)
}

func TestCreateTableDDLPlainNotNullColumn(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows(columnQueryCols).
		AddRow("name", 1, "character varying(255)", true, nil, "", "")
	expectColumnQuery(mock, rows)

	ddl, copyColumns, err := createTableDDL(db, "t")
	assert.NoError(t, err)
	assert.Contains(t, ddl, `"name" character varying(255) NOT NULL`)
	assert.Equal(t, []string{"name"}, copyColumns)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestSetvalStatementEscapesSingleQuotesInSequenceName(t *testing.T) {
	// a sequence name can legally contain a single quote (e.g. a table named
	// `my'thing` quoted as "my'thing"); since setval's first argument is a
	// single-quoted string literal (not an identifier), that quote must be
	// doubled or it terminates the literal early and corrupts the statement.
	got := setvalStatement(`"public"."my'seq"`, 1, true)
	want := `SELECT pg_catalog.setval('"public"."my''seq"', 1, true);`
	if got != want {
		t.Errorf("setvalStatement(...) = %q, want %q", got, want)
	}
}

func TestSequenceRestoreStatementsUseLiveSequenceStateVerbatim(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	ownedSeqRows := sqlmock.NewRows([]string{"seq_name", "col_name"}).
		AddRow("t_id_seq", "id")
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_depend d.*deptype IN`).WillReturnRows(ownedSeqRows)

	// deliberately NOT the MAX(id)+... of any data - proves we read the
	// sequence's own last_value/is_called rather than recomputing from rows,
	// which matters when whereClauseForTables filters out the highest ids.
	seqStateRows := sqlmock.NewRows([]string{"last_value", "is_called"}).
		AddRow(42, true)
	mock.ExpectQuery(`^SELECT last_value, is_called FROM "public"\."t_id_seq"$`).WillReturnRows(seqStateRows)

	statements, err := sequenceRestoreStatements(db, `"public"."t"`)
	assert.NoError(t, err)
	assert.Equal(t, []string{
		`SELECT pg_catalog.setval('"public"."t_id_seq"', 42, true);`,
	}, statements)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetIndexesExcludesConstraintBackedIndexes(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	// Only the plain secondary index should be returned - the query itself
	// excludes the PK index (NOT i.indisprimary) and unique-constraint-backed
	// indexes (NOT EXISTS ... conindid); a wrongly shaped query that forgot
	// either filter would not match this expectation.
	rows := sqlmock.NewRows([]string{"indexdef"}).
		AddRow(`CREATE INDEX t_email_idx ON public.t USING btree (email)`)
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_index i.*NOT i\.indisprimary.*conindid`).
		WillReturnRows(rows)

	indexDefs, err := getIndexes(db, `"public"."t"`)
	assert.NoError(t, err)
	assert.Equal(t, []string{`CREATE INDEX t_email_idx ON public.t USING btree (email);`}, indexDefs)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestConstraintsAreEmittedInPkUniqueCheckFkOrder(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	// pg_constraint's natural query order (contype, conname) is alphabetical
	// (c, f, p, u) - deliberately return them scrambled/alphabetical here to
	// prove the emission order is enforced by our code, not by accident of
	// the query's own ORDER BY.
	rows := sqlmock.NewRows([]string{"conname", "contype", "definition"}).
		AddRow("t_status_check", "c", "CHECK ((status = ANY (ARRAY['a','b'])))").
		AddRow("t_parent_id_fkey", "f", `FOREIGN KEY (parent_id) REFERENCES "public".t_parent(id)`).
		AddRow("t_pkey", "p", "PRIMARY KEY (id)").
		AddRow("t_email_key", "u", "UNIQUE (email)")
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_constraint con.*contype IN`).WillReturnRows(rows)

	constraints, err := getConstraints(db, `"public"."t"`)
	assert.NoError(t, err)

	ordered := orderConstraints(constraints)
	var statements []string
	for _, c := range ordered {
		statements = append(statements, c.alterStatement(`"public"."t"`))
	}

	assert.Equal(t, []string{
		`ALTER TABLE "public"."t" ADD CONSTRAINT "t_pkey" PRIMARY KEY (id);`,
		`ALTER TABLE "public"."t" ADD CONSTRAINT "t_email_key" UNIQUE (email);`,
		`ALTER TABLE "public"."t" ADD CONSTRAINT "t_status_check" CHECK ((status = ANY (ARRAY['a','b'])));`,
		`ALTER TABLE "public"."t" ADD CONSTRAINT "t_parent_id_fkey" FOREIGN KEY (parent_id) REFERENCES "public".t_parent(id);`,
	}, statements)

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTableDDLIdentityColumns(t *testing.T) {
	tests := []struct {
		name         string
		identityKind string
		defaultExpr  interface{}
		want         string
	}{
		{name: "GENERATED ALWAYS AS IDENTITY, no default emitted", identityKind: "a", defaultExpr: nil, want: `"id" integer GENERATED ALWAYS AS IDENTITY`},
		{name: "GENERATED BY DEFAULT AS IDENTITY, default suppressed", identityKind: "d", defaultExpr: "nextval('t_id_seq'::regclass)", want: `"id" integer GENERATED BY DEFAULT AS IDENTITY`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			assert.NoError(t, err, "an error was not expected when opening a stub database connection")
			defer db.Close()

			rows := sqlmock.NewRows(columnQueryCols).
				AddRow("id", 1, "integer", true, tt.defaultExpr, tt.identityKind, "")
			expectColumnQuery(mock, rows)

			ddl, copyColumns, err := createTableDDL(db, "t")
			assert.NoError(t, err)
			assert.Contains(t, ddl, tt.want)
			assert.NotContains(t, ddl, "DEFAULT nextval", "identity columns must not also emit a DEFAULT clause")
			assert.Equal(t, []string{"id"}, copyColumns)

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestCreateTableDDLStoredGeneratedColumnExcludedFromCopy(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows(columnQueryCols).
		AddRow("first_name", 1, "text", false, nil, "", "").
		AddRow("last_name", 2, "text", false, nil, "", "").
		AddRow("full_name", 3, "text", false, "first_name || ' ' || last_name", "", "s")
	expectColumnQuery(mock, rows)

	ddl, copyColumns, err := createTableDDL(db, "t")
	assert.NoError(t, err)
	assert.Contains(t, ddl, `"full_name" text GENERATED ALWAYS AS (first_name || ' ' || last_name) STORED`)
	assert.Equal(t, []string{"first_name", "last_name"}, copyColumns,
		"stored generated columns must be excluded from the COPY column list - Postgres computes them itself")

	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateTableDDLColumnWithDefault(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	rows := sqlmock.NewRows(columnQueryCols).
		AddRow("balance", 1, "numeric(10,2)", false, "0", "", "")
	expectColumnQuery(mock, rows)

	ddl, copyColumns, err := createTableDDL(db, "t")
	assert.NoError(t, err)
	assert.Contains(t, ddl, `"balance" numeric(10,2) DEFAULT 0`)
	assert.Equal(t, []string{"balance"}, copyColumns)

	assert.NoError(t, mock.ExpectationsWereMet())
}
