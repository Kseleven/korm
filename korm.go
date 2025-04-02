package korm

import (
	"context"
	"github.com/jackc/pgx/v5"
)

type DB struct {
	*pgx.Conn
	DBPattern
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
	return &DB{Conn: conn, DBPattern: defaultPattern, tableCache: make(map[string]*Field)}
}

func (s *DB) Close() error {
	return s.Conn.Close(context.Background())
}

func (s *DB) Begin() (Transaction, error) {
	tx, err := s.Conn.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	return DBTx{Tx: tx, Driver: s}, nil
}

func (s *DB) GetDBPattern() DBPattern {
	return s.DBPattern
}

func (s *DB) GetTableCache(name string) (*Field, bool) {
	f, ok := s.tableCache[name]
	return f, ok
}

type DBTx struct {
	pgx.Tx
	Driver
}

func (tx DBTx) Commit() error {
	return tx.Tx.Commit(context.Background())
}

func (tx DBTx) Rollback() error {
	return tx.Tx.Rollback(context.Background())
}

func (tx DBTx) Exec(query string, args ...any) (int64, error) {
	result, err := tx.Tx.Exec(context.Background(), query, args...)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
