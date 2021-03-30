package mysqldump

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/template"
	"time"
)

type tableData struct {
	Name   string
	SQL    string
	Values string
}

type dumpData struct {
	DumpVersion   string
	ServerVersion string
	Tables        []*tableData
	CompleteTime  string
	Database      string
}

const version = "0.2.2"

const tmpl = `-- Go SQL Dump {{ .DumpVersion }}
--
-- ------------------------------------------------------
-- Server version	{{ .ServerVersion }}

CREATE DATABASE IF NOT EXISTS {{ .Database }};
USE {{ .Database }};

/*!40101 SET @OLD_CHARACTER_SET_CLIENT=@@CHARACTER_SET_CLIENT */;
/*!40101 SET @OLD_CHARACTER_SET_RESULTS=@@CHARACTER_SET_RESULTS */;
/*!40101 SET @OLD_COLLATION_CONNECTION=@@COLLATION_CONNECTION */;
/*!40101 SET NAMES utf8 */;
/*!40103 SET @OLD_TIME_ZONE=@@TIME_ZONE */;
/*!40103 SET TIME_ZONE='+00:00' */;
/*!40014 SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0 */;
/*!40014 SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0 */;
/*!40101 SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='NO_AUTO_VALUE_ON_ZERO' */;
/*!40111 SET @OLD_SQL_NOTES=@@SQL_NOTES, SQL_NOTES=0 */;


{{range .Tables}}
--
-- Table structure for table {{ .Name }}
--

DROP TABLE IF EXISTS {{ .Name }};
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;
{{ .SQL }};
/*!40101 SET character_set_client = @saved_cs_client */;
--
-- Dumping data for table {{ .Name }}
--

LOCK TABLES {{ .Name }} WRITE;
/*!40000 ALTER TABLE {{ .Name }} DISABLE KEYS */;
{{ if .Values }}
INSERT INTO {{ .Name }} VALUES {{ .Values }};
{{ end }}
/*!40000 ALTER TABLE {{ .Name }} ENABLE KEYS */;
UNLOCK TABLES;
{{ end }}
-- Dump completed on {{ .CompleteTime }}
`

// Dump dumps one or more tables from a database into a writer
func (d *Dumper) Dump(w io.Writer, db string, tables ...string) error {
	var err error

	// Use the database
	if _, err = d.db.Exec("USE " + db); err != nil {
		return fmt.Errorf("use database: %w", err)
	}

	data := dumpData{
		DumpVersion: version,
		Tables:      make([]*tableData, 0),
	}

	// Get server version
	if data.ServerVersion, err = getServerVersion(d.db); err != nil {
		return err
	}

	// Get sql for each table
	for _, t := range tables {
		if t, err := createTable(d.db, t); err == nil {
			data.Tables = append(data.Tables, t)
		} else {
			return err
		}
	}

	// Set complete time
	data.CompleteTime = time.Now().String()

	// Write dump to file
	t, err := template.New("mysqldump").Parse(tmpl)
	if err != nil {
		return err
	}
	return t.Execute(w, data)
}

// DumpAllTables dumps all tables in a database into a writer
func (d *Dumper) DumpAllTables(w io.Writer, db string) error {
	// Use the database
	if _, err := d.db.Exec("USE " + db); err != nil {
		return fmt.Errorf("use database: %w", err)
	}

	// List tables in the database
	tables, err := d.getTables()
	if err != nil {
		return fmt.Errorf("list tables: %w", err)
	}

	return d.Dump(w, db, tables...)
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

func createTable(db *sql.DB, name string) (*tableData, error) {
	var err error
	t := &tableData{Name: name}

	if t.SQL, err = createTableSQL(db, name); err != nil {
		return nil, err
	}

	if t.Values, err = createTableValues(db, name); err != nil {
		return nil, err
	}

	return t, nil
}

func createTableSQL(db *sql.DB, name string) (string, error) {
	// Get table creation SQL
	var table_return sql.NullString
	var table_sql sql.NullString
	err := db.QueryRow("SHOW CREATE TABLE "+name).Scan(&table_return, &table_sql)

	if err != nil {
		return "", err
	}
	if table_return.String != name {
		return "", errors.New("Returned table is not the same as requested table")
	}

	return table_sql.String, nil
}

func createTableValues(db *sql.DB, name string) (string, error) {
	// Get Data
	rows, err := db.Query("SELECT * FROM " + name)
	if err != nil {
		return "", err
	}
	defer rows.Close()

	// Get columns
	columns, err := rows.Columns()
	if err != nil {
		return "", err
	}
	if len(columns) == 0 {
		return "", errors.New("No columns in table " + name + ".")
	}

	// Read data
	data_text := make([]string, 0)
	for rows.Next() {
		// Init temp data storage

		//ptrs := make([]interface{}, len(columns))
		//var ptrs []interface {} = make([]*sql.NullString, len(columns))

		data := make([]*sql.NullString, len(columns))
		ptrs := make([]interface{}, len(columns))
		for i, _ := range data {
			ptrs[i] = &data[i]
		}

		// Read data
		if err := rows.Scan(ptrs...); err != nil {
			return "", err
		}

		dataStrings := make([]string, len(columns))

		for key, value := range data {
			if value != nil && value.Valid {
				dataStrings[key] = "'" + value.String + "'"
			} else {
				dataStrings[key] = "null"
			}
		}

		data_text = append(data_text, "("+strings.Join(dataStrings, ",")+")")
	}

	return strings.Join(data_text, ","), rows.Err()
}
