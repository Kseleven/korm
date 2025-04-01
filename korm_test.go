package korm

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net"
	"net/netip"
	"testing"
	"time"
)

type User struct {
	ID      int
	Name    string
	Age     int
	IPs     []net.IP
	Subnets []netip.Prefix
	Numbers []int
}

const ConnStr string = "user=test password=test host=192.168.54.137 port=5432 database=test sslmode=disable"

func TestNewDB(t *testing.T) {
	db, err := NewDB(ConnStr)
	require.NoError(t, err)
	defer db.Close()
	//insert
	//insertSql := `INSERT INTO users (id, name, age, ips, subnets, numbers) VALUES(1,'test',18,'{192.168.1.1,10.0.0.1}','{10.0.0.0/24}','{1,2,3}')`
	//_, err = db.Exec(insertSql)
	//require.NoError(t, err)

	rows, err := db.Query(context.Background(), "SELECT * FROM users LIMIT 1")
	require.NoError(t, err)
	defer rows.Close()

	var user User
	require.NoError(t, scanToStruct(rows, &user))
	t.Logf("User: %+v\n", user)
}

type Student struct {
	Id       int
	CreateAt time.Time
	Name     string
	Age      int
	Address  netip.Addr
}

func TestInsertOne(t *testing.T) {
	db, err := NewDB(ConnStr)
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

	assert.NoError(t, db.WithTx(func(tx pgx.Tx) error {
		return db.Insert(tx, student)
	}))
}

func TestInsertMany(t *testing.T) {
	db, err := NewDB(ConnStr)
	require.NoError(t, err)
	require.NoError(t, db.RegisterModels(Student{}))

	var students []*Student
	addr := netip.MustParseAddr("192.168.1.0")
	for i := 0; i < 10; i++ {
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

	assert.NoError(t, db.WithTx(func(tx pgx.Tx) error {
		return db.Insert(tx, students)
	}))
}

func TestQuery(t *testing.T) {
}
