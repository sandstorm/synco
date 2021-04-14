package mysqldump

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"

	binary "github.com/MouseHatGames/go-mysqldump/internal/marshal"
)

const version = "1.0.0"

var comma = []byte{','}
var commaNewline = []byte{',', '\n', '\t'}
var quote = []byte{'\''}

// Dumper represents a database.
type Dumper struct {
	db  *sql.DB
	w   io.Writer
	bin *binary.Writer
}

// NewDumper creates a new dumper instance.
func NewDumper(db *sql.DB, w io.Writer) *Dumper {
	return &Dumper{
		db:  db,
		w:   w,
		bin: binary.NewWriter(w),
	}
}

// Dump dumps one or more tables from a database into a writer.
// If dbName is not empty, a "USE xxx" command will be sent prior to commencing the dump.
func (d *Dumper) Dump(dbName string, tables ...string) error {
	var err error

	if len(tables) == 0 {
		return nil
	}

	// Get server version
	serverVer, err := getServerVersion(d.db)
	if err != nil {
		return err
	}

	if err = d.use(dbName); err != nil {
		return err
	}

	d.bin.WriteFileHeader(&binary.FileHeader{
		ServerVersion: serverVer,
		DatabaseName:  dbName,
		DumpStart:     time.Now().UTC(),
	})

	// Write sql for each table
	for _, t := range tables {
		if err := d.writeTable(t, dbName); err != nil {
			return err
		}
	}

	return nil
}

// DumpAllTables dumps all tables in a database into a writer
// If dbName is not empty, a "USE xxx" command will be sent prior to commencing the dump.
func (d *Dumper) DumpAllTables(dbName string) error {
	if err := d.use(dbName); err != nil {
		return err
	}

	// List tables in the database
	tables, err := d.getTables()
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	return d.Dump(dbName, tables...)
}

func (d *Dumper) use(db string) error {
	if db != "" {
		// Use the database
		if _, err := d.db.Exec("USE " + db); err != nil {
			return fmt.Errorf("use database: %w", err)
		}
	}

	return nil
}

func (d *Dumper) getTables() ([]string, error) {
	tables := make([]string, 0)

	// Get table list
	rows, err := d.db.Query("SHOW TABLES")
	if err != nil {
		return tables, err
	}
	defer rows.Close()

	// Read result
	for rows.Next() {
		var table sql.NullString
		if err := rows.Scan(&table); err != nil {
			return tables, err
		}
		tables = append(tables, table.String)
	}
	return tables, rows.Err()
}

func getServerVersion(db *sql.DB) (string, error) {
	var server_version sql.NullString
	if err := db.QueryRow("SELECT version()").Scan(&server_version); err != nil {
		return "", err
	}
	return server_version.String, nil
}

func (d *Dumper) writeTable(name string, schema string) error {
	var err error

	sql, err := getTableSQL(d.db, name)
	if err != nil {
		return fmt.Errorf("get table SQL: %w", err)
	}

	cols, err := getTableColumns(d.db, name, schema)
	if err != nil {
		return fmt.Errorf("get table columns: %w", err)
	}

	d.bin.WriteTableHeader(&binary.TableHeader{
		Name:      name,
		CreateSQL: sql,
		Columns:   cols,
	})

	if err = d.writeTableValues(name); err != nil {
		return fmt.Errorf("write table rows: %w", err)
	}

	return nil
}

func getTableSQL(db *sql.DB, name string) (string, error) {
	// Get table creation SQL
	var table_return sql.NullString
	var table_sql sql.NullString
	err := db.QueryRow("SHOW CREATE TABLE "+name).Scan(&table_return, &table_sql)

	if err != nil {
		return "", err
	}
	if table_return.String != name {
		return "", errors.New("returned table is not the same as requested table")
	}

	return table_sql.String, nil
}

func getTableColumns(db *sql.DB, table string, schema string) (cols []string, err error) {
	rows, err := db.Query("SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_NAME = ? AND TABLE_SCHEMA = ?", table, schema)
	if err != nil {
		return nil, err
	}

	var column string
	for rows.Next() {
		err = rows.Scan(&column)
		if err != nil {
			return nil, err
		}

		cols = append(cols, column)
	}

	return
}

func (d *Dumper) writeTableValues(name string) error {
	// Get Data
	rows, err := d.db.Query("SELECT * FROM " + name)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Get columns
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return errors.New("no columns in table " + name + ".")
	}

	for rows.Next() {
		if err = d.writeValues(rows, columns); err != nil {
			return fmt.Errorf("write values: %w", err)
		}
	}

	return nil
}

func (d *Dumper) writeValues(rows *sql.Rows, columns []string) error {
	data := make([]*string, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range data {
		ptrs[i] = &data[i]
	}

	// Read data
	if err := rows.Scan(ptrs...); err != nil {
		return err
	}

	return d.bin.WriteRowData(data)
}
