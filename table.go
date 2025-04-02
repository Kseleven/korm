package korm

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/netip"
	"reflect"
	"strings"
	"time"
)

func (s *DB) RegisterModels(models ...any) error {
	for _, model := range models {
		sqlList, err := s.genCreateTableSql(model)
		if err != nil {
			return err
		}

		for _, sql := range sqlList {
			fmt.Printf("create table sql:%s\n", sql)
			if _, err := s.Conn.Exec(context.Background(), sql); err != nil {
				return fmt.Errorf("register table %s failed: %w", reflect.TypeOf(model).Name(), err)
			}
		}
	}
	return nil
}

func (s *DB) genCreateTableSql(model any) ([]string, error) {
	t := reflect.TypeOf(model)
	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("model must be a struct")
	}
	tableName := s.TableName(t.Name())
	compositeIdxMap := make(map[string][]string)
	unique := make(map[string]struct{})
	columns, colTypes, createIdxSql, err := s.parseFields(tableName, t, unique, compositeIdxMap)
	if err != nil {
		return nil, err
	}
	field := newField(tableName)
	field.addColumns(columns)
	s.tableCache[field.TableName] = field

	colSql := make([]string, len(columns))
	for i, column := range columns {
		colSql[i] = fmt.Sprintf("%s %s", column, colTypes[i])
	}
	createTableSql := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n%s\n);",
		tableName, strings.Join(colSql, ",\n"))
	for indexName, fields := range compositeIdxMap {
		indexSQL := fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s);", tableName, indexName, tableName, strings.Join(fields, ", "))
		createIdxSql = append(createIdxSql, indexSQL)
	}

	return append([]string{createTableSql}, createIdxSql...), nil
}

func (s *DB) getTableName(model any) (string, error) {
	t := reflect.TypeOf(model)
	if t.Kind() != reflect.Struct {
		return "", fmt.Errorf("model must be a struct")
	}
	return s.TableName(t.Name()), nil
}

func (s *DB) parseFields(tableName string, t reflect.Type, unique map[string]struct{}, compositeIdxMap map[string][]string) (
	columns []string, columnTypes []string, createIdxSql []string, err error) {
	columns = make([]string, 0, t.NumField())
	columnTypes = make([]string, 0, t.NumField())
	createIdxSql = make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if isFieldEmbed(field) {
			col, colTypes, idxSql, err := s.parseFields(tableName, field.Type, unique, compositeIdxMap)
			if err != nil {
				return nil, nil, nil, err
			} else {
				columns = append(columns, col...)
				columnTypes = append(columnTypes, colTypes...)
				createIdxSql = append(createIdxSql, idxSql...)
			}
		} else {
			col, colTypes, indexSQL, compositeIndex, err := s.genColumnSql(tableName, field)
			if err != nil {
				return nil, nil, nil, err
			}
			if len(col) == 0 {
				continue
			}
			columns = append(columns, col)
			columnTypes = append(columnTypes, colTypes)
			if len(indexSQL) != 0 {
				createIdxSql = append(createIdxSql, indexSQL)
			}
			if len(compositeIndex) != 0 {
				compositeIdxMap[compositeIndex] = append(compositeIdxMap[compositeIndex], s.ColumnName(field.Name))
			}
		}

		columnName := s.DBPattern.ColumnName(field.Name)
		if _, ok := unique[columnName]; !ok {
			unique[columnName] = struct{}{}
		} else {
			return nil, nil, nil, fmt.Errorf("column %s already exists", columnName)
		}
	}
	return columns, columnTypes, createIdxSql, nil
}

func isFieldEmbed(field reflect.StructField) bool {
	if field.Anonymous {
		return true
	}
	if tag := field.Tag.Get("db"); strings.Contains(tag, "embed") {
		return true
	}
	return false
}

func (s *DB) genColumnSql(tableName string, field reflect.StructField) (col string, colType string, indexSQL string, compositeIndex string, err error) {
	name := s.DBPattern.ColumnName(field.Name)
	tag := field.Tag.Get("db")
	var ukIndex string
	if len(tag) > 0 {
		if strings.Contains(tag, "-") {
			return
		}
		if strings.Contains(tag, "pk") {
			ukIndex += " PRIMARY KEY"
		}
		if strings.Contains(tag, "uk") {
			ukIndex += " UNIQUE"
		}
		if strings.Contains(tag, "notNull") {
			ukIndex += " NOT NULL"
		}
		if strings.Contains(tag, "index") {
			if strings.Contains(tag, "index=") {
				compositeIndex = extractIndexName(tag)
			} else {
				indexSQL = fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s (%s);", tableName, name, tableName, name)
			}
		}
	}

	tpy := field.Type
	dbType, err := goTypeToPostgresType(tpy)
	if err != nil {
		return "", "", "", "", err
	}

	if dbType == "JSONB" && indexSQL != "" {
		indexSQL = fmt.Sprintf("CREATE INDEX IF NOT EXISTS idx_%s_%s ON %s USING GIN (%s);", tableName, name, tableName, name)
	}
	col = name
	colType = dbType + ukIndex
	return
}

func extractIndexName(tags string) string {
	parts := strings.Split(tags, "index=")
	if len(parts) > 1 {
		return strings.TrimSpace(parts[1])
	}
	return ""
}

func goTypeToPostgresType(goType reflect.Type) (string, error) {
	switch goType.Kind() {
	case reflect.Int16:
		return "SMALLINT", nil
	case reflect.Int, reflect.Int32:
		return "INTEGER", nil
	case reflect.Int64:
		return "BIGINT", nil
	case reflect.Float32:
		return "FLOAT4", nil
	case reflect.Uint, reflect.Uint64, reflect.Float64:
		return "NUMERIC", nil
	case reflect.String:
		return "TEXT", nil
	case reflect.Bool:
		return "BOOLEAN", nil
	case reflect.Map:
		return "JSONB", nil
	case reflect.Struct, reflect.Ptr:
		t := goType
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		switch t {
		case reflect.TypeOf(time.Time{}):
			return "TIMESTAMP", nil
		case reflect.TypeOf(net.IP{}), reflect.TypeOf(netip.Addr{}):
			return "CIDR", nil
		case reflect.TypeOf(net.IPNet{}), reflect.TypeOf(netip.Prefix{}):
			return "INET", nil
		default:
			return "", fmt.Errorf("unknown type :%s", t.String())
		}
	case reflect.Slice, reflect.Array:
		elemKind := goType.Elem().Kind()
		switch elemKind {
		case reflect.Uint8:
			if goType.Kind() == reflect.TypeOf(json.RawMessage{}).Kind() {
				return "JSONB", nil
			}
			return "BYTEA", nil
		case reflect.String:
			return "TEXT[]", nil
		case reflect.Int16:
			return "SMALLINT[]", nil
		case reflect.Int, reflect.Int32:
			return "INTEGER[]", nil
		case reflect.Int64:
			return "BIGINT[]", nil
		case reflect.Float32:
			return "FLOAT4[]", nil
		case reflect.Uint, reflect.Uint64, reflect.Float64:
			return "NUMERIC[]", nil
		case reflect.Struct, reflect.Ptr:
			t := goType.Elem()
			if t.Kind() == reflect.Ptr {
				t = t.Elem()
			}
			switch t {
			case reflect.TypeOf(net.IP{}), reflect.TypeOf(netip.Addr{}):
				return "CIDR[]", nil
			case reflect.TypeOf(net.IPNet{}), reflect.TypeOf(netip.Prefix{}):
				return "INET[]", nil
			default:
				return "", fmt.Errorf("unsupported type %s", t.String())
			}
		default:
			return "", fmt.Errorf("unsupported type %s", goType.Kind().String())
		}
	default:
		return "", fmt.Errorf("unsupported type %s", goType.Kind().String())
	}
}
