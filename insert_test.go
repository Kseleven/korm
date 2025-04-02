package korm

import (
	"fmt"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type Student struct {
	Id       int
	CreateAt time.Time
	Name     string
	Age      int
	Address  netip.Addr
}

func TestInsertOne(t *testing.T) {
	connStr, err := readEnv()
	require.NoError(t, err)
	db, err := NewDB(connStr)
	require.NoError(t, err)
	require.NoError(t, db.RegisterModels(Student{}))

	addr := netip.MustParseAddr("192.168.1.0")
	addr = addr.Next()
	student := &Student{
		Id:       -1,
		CreateAt: time.Now(),
		Name:     fmt.Sprintf("name%d", 0),
		Age:      0,
		Address:  addr,
	}

	assert.NoError(t, WithTx(db, func(tx Transaction) error {
		return tx.Insert(student)
	}))
}

func TestInsertMany(t *testing.T) {
	connStr, err := readEnv()
	require.NoError(t, err)
	db, err := NewDB(connStr)
	require.NoError(t, err)
	require.NoError(t, db.RegisterModels(Student{}))

	var students []*Student
	addr := netip.MustParseAddr("192.168.1.0")
	// 10000 --> insert 100ms
	// 100000 --> insert 600ms
	// 1000000 --> insert 2.13s
	for i := 0; i < 10000; i++ {
		addr = addr.Next()
		student := &Student{
			Id:       i,
			CreateAt: time.Now(),
			Name:     fmt.Sprintf("name%d", i),
			Age:      i,
			Address:  addr,
		}
		students = append(students, student)
	}

	assert.NoError(t, WithTx(db, func(tx Transaction) error {
		if _, err := tx.Exec("DELETE FROM student"); err != nil {
			return fmt.Errorf("delete student failed: %w", err)
		}
		return tx.Insert(students)
	}))
}
