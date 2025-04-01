package korm

import (
	"context"
	"fmt"
	"reflect"
	"strings"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	*pgx.Conn
	DBNamePattern
	tableCache map[string]*Field
}

func NewDB(connStr string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), connStr)
	if err != nil {
		return nil, err
	}

	return initDB(conn), nil
}

func initDB(conn *pgx.Conn) *DB {
	return &DB{Conn: conn, DBNamePattern: defaultPattern, tableCache: make(map[string]*Field)}
}

func (s *DB) Close() error {
	return s.Conn.Close(context.Background())
}

func (s *DB) WithTx(f func(tx pgx.Tx) error) error {
	tx, err := s.Conn.Begin(context.Background())
	if err != nil {
		return err
	}

	if err = f(tx); err != nil {
		tx.Rollback(context.Background())
		return err
	}
	return tx.Commit(context.Background())
}

func (s *DB) Insert(tx pgx.Tx, data any) error {
	v := reflect.ValueOf(data)

	var rows [][]interface{}
	var field *Field
	var ok bool
	var err error
	if v.Kind() == reflect.Slice {
		field, rows, err = s.buildInsertRows(v)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
	} else if v.Kind() == reflect.Ptr {
		t := v.Type().Elem()
		if field, ok = s.tableCache[s.DBNamePattern.TableName(t.Name())]; !ok {
			return fmt.Errorf("table %s not registered", t.Name())
		}
		rows = make([][]interface{}, 1)
		row, err := s.buildInsertRow(field, v)
		if err != nil {
			return err
		} else {
			rows[0] = row
		}
	} else {
		return fmt.Errorf("data must be a slice or pointer")
	}

	copySQL := fmt.Sprintf("COPY %s (%s) FROM STDIN", field.TableName, strings.Join(field.Columns, ", "))
	fmt.Println(copySQL, rows)
	_, err = tx.CopyFrom(context.Background(), pgx.Identifier{field.TableName}, field.Columns, pgx.CopyFromRows(rows))
	return err
}

func (s *DB) buildInsertRows(v reflect.Value) (*Field, [][]interface{}, error) {
	if v.Len() == 0 {
		return nil, nil, nil
	}
	t := v.Index(0).Type()
	if t.Kind() != reflect.Ptr {
		return nil, nil, fmt.Errorf("data must be a pointer")
	}
	t = t.Elem()

	field, ok := s.tableCache[s.DBNamePattern.TableName(t.Name())]
	if !ok {
		return nil, nil, fmt.Errorf("table %s not registered", t.Name())
	}

	rows := make([][]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		row, err := s.buildInsertRow(field, v.Index(i))
		if err != nil {
			return nil, nil, err
		}
		rows[i] = row
	}
	return field, rows, nil
}

func (s *DB) buildInsertRow(field *Field, v reflect.Value) ([]interface{}, error) {
	v = v.Elem()
	row := make([]interface{}, 0, len(field.Columns))
	for j := 0; j < v.NumField(); j++ {
		f := v.Field(j)
		vName := s.DBNamePattern.ColumnName(v.Type().Field(j).Name)
		if _, ok := field.ColumnMap[vName]; !ok {
			continue
		}
		row = append(row, f.Interface())
	}
	return row, nil
}
