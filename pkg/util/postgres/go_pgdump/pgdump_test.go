package go_pgdump

import (
	"bytes"
	"strings"
	"testing"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

// TestDumpOrdersOutputIntoSchemaThenDataThenConstraintsThenSequences is the
// package's own integration test: it stitches together every catalog/copy
// piece built in the earlier slices for a small two-table schema (t_child
// has an FK to t_parent) and asserts the complete output file ordering:
// both DROP+CREATE TABLEs, then both tables' COPY blocks, then
// PK -> UNIQUE -> secondary indexes -> CHECK -> FK (deferred to last so
// referenced tables/rows already exist), then sequence setval restoration.
func TestDumpOrdersOutputIntoSchemaThenDataThenConstraintsThenSequences(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err, "an error was not expected when opening a stub database connection")
	defer db.Close()

	mock.ExpectBegin()
	mock.ExpectExec(`SET TIME ZONE 'UTC'`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SET extra_float_digits = 3`).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(`SET bytea_output = 'hex'`).WillReturnResult(sqlmock.NewResult(0, 0))

	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_class c.*relkind = 'r'`).
		WillReturnRows(sqlmock.NewRows([]string{"relname"}).
			AddRow("t_child").
			AddRow("t_parent"))

	// Phase 1: column introspection (order follows getTables: t_child, t_parent)
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_attribute a.*attisdropped`).
		WithArgs(`"public"."t_child"`).
		WillReturnRows(sqlmock.NewRows(columnQueryCols).
			AddRow("id", 1, "integer", true, nil, "a", "").
			AddRow("parent_id", 2, "integer", true, nil, "", "").
			AddRow("status", 3, "text", false, nil, "", ""))
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_attribute a.*attisdropped`).
		WithArgs(`"public"."t_parent"`).
		WillReturnRows(sqlmock.NewRows(columnQueryCols).
			AddRow("id", 1, "integer", true, nil, "a", "").
			AddRow("name", 2, "text", true, nil, "", ""))

	// Phase 2: data
	mock.ExpectQuery(`^SELECT "id"::text, "parent_id"::text, "status"::text FROM "public"\."t_child" WHERE TRUE$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id", "status"}).AddRow("1", "1", "ok"))
	mock.ExpectQuery(`^SELECT "id"::text, "name"::text FROM "public"\."t_parent" WHERE TRUE$`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "name"}).AddRow("1", "alpha"))

	// Phase 3: constraints + indexes (t_child then t_parent)
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_constraint con.*contype IN`).
		WithArgs(`"public"."t_child"`).
		WillReturnRows(sqlmock.NewRows([]string{"conname", "contype", "definition"}).
			AddRow("t_child_pkey", "p", "PRIMARY KEY (id)").
			AddRow("t_child_parent_id_fkey", "f", `FOREIGN KEY (parent_id) REFERENCES "public".t_parent(id)`).
			AddRow("t_child_status_check", "c", "CHECK ((status IS NOT NULL))"))
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_index i.*NOT i\.indisprimary`).
		WithArgs(`"public"."t_child"`).
		WillReturnRows(sqlmock.NewRows([]string{"indexdef"}))

	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_constraint con.*contype IN`).
		WithArgs(`"public"."t_parent"`).
		WillReturnRows(sqlmock.NewRows([]string{"conname", "contype", "definition"}).
			AddRow("t_parent_pkey", "p", "PRIMARY KEY (id)").
			AddRow("t_parent_name_key", "u", "UNIQUE (name)"))
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_index i.*NOT i\.indisprimary`).
		WithArgs(`"public"."t_parent"`).
		WillReturnRows(sqlmock.NewRows([]string{"indexdef"}).
			AddRow("CREATE INDEX t_parent_name_idx ON public.t_parent USING btree (name)"))

	// Phase 4: sequence restoration (t_child then t_parent)
	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_depend d.*deptype IN`).
		WithArgs(`"public"."t_child"`).
		WillReturnRows(sqlmock.NewRows([]string{"seq_name", "col_name"}).AddRow("t_child_id_seq", "id"))
	mock.ExpectQuery(`^SELECT last_value, is_called FROM "public"\."t_child_id_seq"$`).
		WillReturnRows(sqlmock.NewRows([]string{"last_value", "is_called"}).AddRow(5, true))

	mock.ExpectQuery(`(?s)FROM pg_catalog\.pg_depend d.*deptype IN`).
		WithArgs(`"public"."t_parent"`).
		WillReturnRows(sqlmock.NewRows([]string{"seq_name", "col_name"}).AddRow("t_parent_id_seq", "id"))
	mock.ExpectQuery(`^SELECT last_value, is_called FROM "public"\."t_parent_id_seq"$`).
		WillReturnRows(sqlmock.NewRows([]string{"last_value", "is_called"}).AddRow(2, true))

	mock.ExpectRollback()

	var buf bytes.Buffer
	dumper := NewDumper(db, &buf)
	assert.NoError(t, dumper.Dump())
	assert.NoError(t, mock.ExpectationsWereMet())

	out := buf.String()
	find := func(substr string) int {
		i := strings.Index(out, substr)
		if i == -1 {
			t.Fatalf("expected output to contain %q, got:\n%s", substr, out)
		}
		return i
	}

	dropChild := find(`DROP TABLE IF EXISTS "public"."t_child" CASCADE;`)
	dropParent := find(`DROP TABLE IF EXISTS "public"."t_parent" CASCADE;`)
	copyChild := find(`COPY "public"."t_child"`)
	copyParent := find(`COPY "public"."t_parent"`)
	pkChild := find(`ADD CONSTRAINT "t_child_pkey" PRIMARY KEY`)
	pkParent := find(`ADD CONSTRAINT "t_parent_pkey" PRIMARY KEY`)
	uniqueParent := find(`ADD CONSTRAINT "t_parent_name_key" UNIQUE`)
	indexParent := find(`CREATE INDEX t_parent_name_idx`)
	checkChild := find(`ADD CONSTRAINT "t_child_status_check" CHECK`)
	fkChild := find(`ADD CONSTRAINT "t_child_parent_id_fkey" FOREIGN KEY`)
	setvalChild := find(`setval('"public"."t_child_id_seq"'`)
	setvalParent := find(`setval('"public"."t_parent_id_seq"'`)

	assert.True(t, dropChild < copyChild && dropParent < copyChild, "both DROP/CREATE TABLEs must precede any COPY")
	assert.True(t, copyChild < pkChild && copyParent < pkChild, "COPY data must precede constraints")
	assert.True(t, pkChild < uniqueParent, "PK constraints must precede UNIQUE constraints")
	assert.True(t, uniqueParent < indexParent, "UNIQUE constraints must precede secondary indexes")
	assert.True(t, indexParent < checkChild, "secondary indexes must precede CHECK constraints")
	assert.True(t, checkChild < fkChild, "CHECK constraints must precede FOREIGN KEY constraints")
	assert.True(t, fkChild < setvalChild && fkChild < setvalParent, "FOREIGN KEY constraints must precede sequence restoration")
	_ = pkParent
}
