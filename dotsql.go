// Package dotsql provides a way to separate your code from SQL queries.
//
// It is not an ORM, it is not a query builder.
// Dotsql is a library that helps you keep sql files in one place and use it with ease.
//
// For more usage examples see https://github.com/gchaincl/dotsql
package dotsql

import (
	"bufio"
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"os"
	"strings"
)

// Preparer is an interface used by Prepare.
type Preparer interface {
	Prepare(query string) (*sql.Stmt, error)
}

// Queryer is an interface used by Query.
type Queryer interface {
	Query(query string, args ...interface{}) (*sql.Rows, error)
}

// QueryRower is an interface used by QueryRow
type QueryRower interface {
	QueryRow(query string, args ...interface{}) *sql.Row
}

// Execer is an interface used by Exec.
type Execer interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

// DotSql represents a dotSQL queries holder.
type DotSql struct {
	queries map[string]string
}

func (d DotSql) lookupQuery(name string) (query string, err error) {
	query, ok := d.queries[name]
	if !ok {
		err = fmt.Errorf("dotsql: '%s' could not be found", name)
	}

	return
}

// Query is a wrapper for database/sql's Prepare(), using dotsql named query.
func (d DotSql) Prepare(db Preparer, name string) (*sql.Stmt, error) {
	query, err := d.lookupQuery(name)
	if err != nil {
		return nil, err
	}

	return db.Prepare(query)
}

// Query is a wrapper for database/sql's Query(), using dotsql named query.
func (d DotSql) Query(db Queryer, name string, args ...interface{}) (*sql.Rows, error) {
	query, err := d.lookupQuery(name)
	if err != nil {
		return nil, err
	}

	return db.Query(query, args...)
}

// QueryRow is a wrapper for database/sql's QueryRow(), using dotsql named query.
func (d DotSql) QueryRow(db QueryRower, name string, args ...interface{}) (*sql.Row, error) {
	query, err := d.lookupQuery(name)
	if err != nil {
		return nil, err
	}

	return db.QueryRow(query, args...), nil
}

// Exec is a wrapper for database/sql's Exec(), using dotsql named query.
func (d DotSql) Exec(db Execer, name string, args ...interface{}) (sql.Result, error) {
	query, err := d.lookupQuery(name)
	if err != nil {
		return nil, err
	}

	return db.Exec(query, args...)
}

// Raw returns the query, everything after the --name tag
func (d DotSql) Raw(name string) (string, error) {
	return d.lookupQuery(name)
}

// QueryMap returns a map[string]string of loaded queries
func (d DotSql) QueryMap() map[string]string {
	return d.queries
}

// Load imports sql queries from any io.Reader.
func Load(r io.Reader) (*DotSql, error) {
	scanner := &Scanner{}
	queries := scanner.Run(bufio.NewScanner(r))

	dotsql := &DotSql{
		queries: queries,
	}

	return dotsql, nil
}

// LoadFromFile imports SQL queries from the file.
func LoadFromFile(sqlFile string) (*DotSql, error) {
	f, err := os.Open(sqlFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return Load(f)
}

// LoadFromFileReplace imports SQL queries from the file.
// It will search for string 'old' and replace it with 'new'.
func LoadFromFileReplace(sqlFile, old, new string) (*DotSql, error) {
	f, err := os.Open(sqlFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := &Scanner{}
	queries := scanner.Run(bufio.NewScanner(f))

	dotsql := &DotSql{
		queries: queries,
	}

	for k, v := range queries {
		dotsql.queries[k] = strings.Replace(v, old, new, -1)
	}

	return dotsql, nil
}

// LoadFromStringReplace imports SQL queries from the string.
// It will search for string 'old' and replace it with 'new'.
func LoadFromStringReplace(sql, old, new string) (*DotSql, error) {
	o := strings.Split(old, ",")
	n := strings.Split(new, ",")

	if len(o) != len(n) {
		err := fmt.Errorf("dotsql: amount of %d old strings is not equal to %d new strings", len(o), len(n))
		return nil, err
	}

	for i := 0; i < len(n); i++ {
		sql = strings.Replace(sql, o[i], n[i], -1)
	}

	buf := bytes.NewBufferString(sql)

	scanner := &Scanner{}
	queries := scanner.Run(bufio.NewScanner(buf))

	dotsql := &DotSql{
		queries: queries,
	}

	for k, v := range queries {
		dotsql.queries[k] = v
	}

	return dotsql, nil
}

// LoadFromString imports SQL queries from the string.
func LoadFromString(sql string) (*DotSql, error) {
	buf := bytes.NewBufferString(sql)
	return Load(buf)
}

// Merge takes one or more *DotSql and merge its queries
// It's in-order, so the last source will override queries with the same name
// in the previous arguments if any.
func Merge(dots ...*DotSql) *DotSql {
	queries := make(map[string]string)

	for _, dot := range dots {
		for k, v := range dot.QueryMap() {
			queries[k] = v
		}
	}

	return &DotSql{
		queries: queries,
	}
}
