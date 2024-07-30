// readdbcfg.go
package stmysqlgen

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"
)

type MSColumn struct {
	Name string
	Type string
	Key  string
	Null bool
}
type MSTable struct {
	Name      string
	DB        string
	Cols      []MSColumn
	CreateCmd string
	PriKeyNum int
	PriKey    MSColumn
}

var (
	Tables []*MSTable
)

func showDB(user, pwd, ip string, port int) ([]string, error) {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s:%s@tcp(%s:%d)/", user, pwd, ip, port)
	db, err := sql.Open("mysql", builder.String())
	if err != nil {
		return nil, err
	}
	defer db.Close()

	res, err := db.Query("show databases")
	if err != nil {
		return nil, err
	}

	var dbs []string
	for res.Next() {
		var name string
		res.Scan(&name)
		dbs = append(dbs, name)
	}
	return dbs, nil
}

func readDB(user, pwd, ip string, port int, dbname string) error {
	var builder strings.Builder
	fmt.Fprintf(&builder, "%s:%s@tcp(%s:%d)/%s", user, pwd, ip, port, dbname)
	db, err := sql.Open("mysql", builder.String())
	if err != nil {
		return err
	}
	defer db.Close()
	return readTable(db, dbname)
}

func readTable(db *sql.DB, dbname string) error {
	res, err := db.Query("show tables")
	if err != nil {
		return err
	}
	for res.Next() {
		var name string
		res.Scan(&name)
		table := &MSTable{name, dbname, nil, "", 0, MSColumn{}}
		err = readColumn(db, table)
		if err != nil {
			return err
		}
		err = readCreateCmd(db, table)
		if err != nil {
			return err
		}
		Tables = append(Tables, table)
	}
	return nil
}

func readColumn(db *sql.DB, table *MSTable) error {
	res, err := db.Query("SHOW COLUMNS FROM " + table.Name)
	if err != nil {
		return err
	}
	for res.Next() {
		var field, typ, null, key, extra string
		var defau sql.RawBytes
		err = res.Scan(&field, &typ, &null, &key, &defau, &extra)
		if err != nil {
			return err
		}
		col := MSColumn{field, typ, key, false}
		if null == "YES" {
			col.Null = true
		}
		if key == "PRI" {
			table.PriKeyNum++
			table.PriKey = col
		}
		table.Cols = append(table.Cols, col)
	}
	return nil
}

func readCreateCmd(db *sql.DB, table *MSTable) error {
	res, err := db.Query("SHOW CREATE TABLE " + table.Name)
	if err != nil {
		return err
	}
	for res.Next() {
		var name, createstr string
		err = res.Scan(&name, &createstr)
		if err != nil {
			return err
		}
		table.CreateCmd = strings.Replace(createstr, "`", "", -1)
	}
	return nil
}
