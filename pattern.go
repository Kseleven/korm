package korm

import (
	"bytes"
	"strings"
)

type DBPattern interface {
	TableName(string) string
	ColumnName(string) string
}

var defaultPattern snakePattern

type snakePattern struct{}

func (snakePattern) TableName(t string) string {
	return ToSnake(t)
}

func (snakePattern) ColumnName(t string) string {
	return ToSnake(t)
}

func ToSnake(s string) string {
	buf := bytes.NewBufferString("")
	for i, v := range s {
		if i > 0 && v >= 'A' && v <= 'Z' {
			buf.WriteRune('_')
		}
		buf.WriteRune(v)
	}
	return strings.ToLower(buf.String())
}

func ToUpperCamel(s string) string {
	buf := bytes.NewBufferString("")
	for _, v := range strings.Split(s, "_") {
		if len(v) > 0 {
			buf.WriteString(strings.ToUpper(v[:1]))
			buf.WriteString(v[1:])
		}
	}
	return buf.String()
}
