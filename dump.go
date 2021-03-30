package mysqldump

import (
	"database/sql"
	"errors"
	"fmt"
	"io"
	"time"
)

const version = "0.2.2"

var comma = []byte{','}

// Dump dumps one or more tables from a database into a writer
func (d *Dumper) Dump(w io.Writer, db string, tables ...string) error {
	var err error

	if len(tables) == 0 {
		return errors.New("no tables to back up")
	}

	// Use the database
	if _, err = d.db.Exec("USE " + db); err != nil {
		return fmt.Errorf("use database: %w", err)
	}

	// Get server version
	serverVer, err := getServerVersion(d.db)
	if err != nil {
		return err
	}

	fmt.Fprintf(w, `-- Go SQL Dump %s
--
-- ------------------------------------------------------
-- Server version	%s

CREATE DATABASE IF NOT EXISTS %s;
USE %[3]s;

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

`, version, serverVer, db)

	// Write sql for each table
	for _, t := range tables {
		if err := writeTable(w, d.db, t); err != nil {
			return err
		}
	}

	fmt.Fprintf(w, "\n-- Dump completed on %s", time.Now())

	return nil
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

func writeTable(w io.Writer, db *sql.DB, name string) error {
	var err error

	fmt.Fprintf(w, `--
-- Table structure for table %[1]s
--

DROP TABLE IF EXISTS %[1]s;
/*!40101 SET @saved_cs_client     = @@character_set_client */;
/*!40101 SET character_set_client = utf8 */;

`, name)

	if err = writeTableSQL(w, db, name); err != nil {
		return fmt.Errorf("write table SQL: %w", err)
	}

	fmt.Fprintf(w, `
/*!40101 SET character_set_client = @saved_cs_client */;
--
-- Dumping data for table %s
--

LOCK TABLES %[1]s WRITE;
/*!40000 ALTER TABLE %[1]s DISABLE KEYS */;
`, name)

	if err = writeTableValues(w, db, name); err != nil {
		return fmt.Errorf("write table rows: %w", err)
	}

	fmt.Fprintf(w, `
/*!40000 ALTER TABLE %s ENABLE KEYS */;
UNLOCK TABLES;
`, name)

	return nil
}

func writeTableSQL(w io.Writer, db *sql.DB, name string) error {
	// Get table creation SQL
	var table_return sql.NullString
	var table_sql sql.NullString
	err := db.QueryRow("SHOW CREATE TABLE "+name).Scan(&table_return, &table_sql)

	if err != nil {
		return err
	}
	if table_return.String != name {
		return errors.New("returned table is not the same as requested table")
	}

	io.WriteString(w, table_sql.String)

	return nil
}

func writeTableValues(w io.Writer, db *sql.DB, name string) error {
	// Get Data
	rows, err := db.Query("SELECT * FROM " + name)
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

	// Read first row
	if rows.Next() {
		fmt.Fprintf(w, "INSERT INTO %s VALUES ", name)

		if err = writeValues(w, rows, columns); err != nil {
			return fmt.Errorf("write values: %w", err)
		}

		// Read the remaining rows
		for rows.Next() {
			w.Write(comma)

			if err = writeValues(w, rows, columns); err != nil {
				return fmt.Errorf("write values: %w", err)
			}
		}
	}

	return nil
}

func writeValues(w io.Writer, rows *sql.Rows, columns []string) error {
	data := make([]*sql.NullString, len(columns))
	ptrs := make([]interface{}, len(columns))
	for i := range data {
		ptrs[i] = &data[i]
	}

	// Read data
	if err := rows.Scan(ptrs...); err != nil {
		return err
	}

	w.Write([]byte{'('})

	for i, v := range data {
		if v != nil && v.Valid {
			fmt.Fprint(w, "'", v.String, "'")
		} else {
			fmt.Fprint(w, "null")
		}

		if i < len(data)-1 {
			w.Write(comma)
		}
	}

	w.Write([]byte{')'})
	return nil
}
