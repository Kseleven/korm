package korm

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
)

func (tx DBTx) Select(out any, query string, args ...any) error {
	rows, err := tx.Tx.Query(context.Background(), query, args...)
	if err != nil {
		return err
	}

	return tx.scanRows(rows, out)
}

func (tx DBTx) scanRows(rows pgx.Rows, out interface{}) error {
	v := reflect.ValueOf(out)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Slice {
		return fmt.Errorf("out must be a pointer to a slice")
	}

	v = v.Elem()
	t := v.Type().Elem()
	if t.Kind() != reflect.Ptr {
		return fmt.Errorf("out element must be a pointer")
	}
	t = t.Elem()
	fieldMap := make(map[string]string, t.NumField())
	s := tx.Driver
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.Tag.Get("db") == "-" {
			continue
		}
		if isFieldEmbed(field) {
			for k := 0; k < field.Type.NumField(); k++ {
				fv := field.Type.Field(k)
				fieldMap[s.GetDBPattern().ColumnName(fv.Name)] = fv.Name
			}
			continue
		}
		fieldMap[s.GetDBPattern().ColumnName(field.Name)] = field.Name
	}

	fds := rows.FieldDescriptions()
	for rows.Next() {
		e := reflect.New(t)
		scanTargets := make([]any, 0, len(fds))
		for _, fd := range fds {
			if colName, ok := fieldMap[fd.Name]; ok {
				scanTargets = append(scanTargets, e.Elem().FieldByName(colName).Addr().Interface())
			}
		}
		if err := rows.Scan(scanTargets...); err != nil {
			return err
		}
		v.Set(reflect.Append(v, e))
	}

	return nil
}
