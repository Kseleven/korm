package korm

import (
	"context"
	"fmt"
	"reflect"

	"github.com/jackc/pgx/v5"
)

func (tx DBTx) Insert(data any) error {
	v := reflect.ValueOf(data)

	var rows [][]interface{}
	var field *Field
	var ok bool
	var err error
	if v.Kind() == reflect.Slice {
		field, rows, err = tx.buildInsertRows(v)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}
	} else if v.Kind() == reflect.Ptr {
		t := v.Type().Elem()
		name := tx.GetDBPattern().TableName(t.Name())
		if field, ok = tx.GetTableCache(name); !ok {
			return fmt.Errorf("table %s not registered", t.Name())
		}
		rows = make([][]interface{}, 1)
		row, err := tx.buildInsertRow(field, t, v)
		if err != nil {
			return err
		} else {
			rows[0] = row
		}
	} else {
		return fmt.Errorf("data must be a slice or pointer")
	}

	_, err = tx.CopyFrom(context.Background(), pgx.Identifier{field.TableName}, field.Columns, pgx.CopyFromRows(rows))
	return err
}

func (tx DBTx) buildInsertRows(v reflect.Value) (*Field, [][]interface{}, error) {
	if v.Len() == 0 {
		return nil, nil, nil
	}
	t := v.Index(0).Type()
	if t.Kind() != reflect.Ptr {
		return nil, nil, fmt.Errorf("data must be a pointer")
	}
	t = t.Elem()

	field, ok := tx.GetTableCache(tx.GetDBPattern().TableName(t.Name()))
	if !ok {
		return nil, nil, fmt.Errorf("table %s not registered", t.Name())
	}

	rows := make([][]interface{}, v.Len())
	for i := 0; i < v.Len(); i++ {
		row, err := tx.buildInsertRow(field, t, v.Index(i))
		if err != nil {
			return nil, nil, err
		}
		rows[i] = row
	}
	return field, rows, nil
}

func (tx DBTx) buildInsertRow(field *Field, t reflect.Type, v reflect.Value) ([]interface{}, error) {
	v = v.Elem()
	row := make([]interface{}, 0, len(field.Columns))
	for j := 0; j < v.NumField(); j++ {
		f := v.Field(j)
		tf := t.Field(j)
		if isFieldEmbed(tf) {
			for k := 0; k < f.NumField(); k++ {
				row = append(row, f.Field(k).Interface())
			}
			continue
		}
		vName := tx.GetDBPattern().ColumnName(v.Type().Field(j).Name)
		if _, ok := field.ColumnMap[vName]; !ok {
			continue
		}
		row = append(row, f.Interface())
	}
	return row, nil
}
