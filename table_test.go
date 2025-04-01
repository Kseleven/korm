package korm

import (
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/netip"
	"testing"
	"time"
)

type Model struct {
	Id       int64
	CreateAt time.Time
	Name     string
	Age      int
	Address  netip.Addr
}

type IndexModel struct {
	Id           int64           `db:"pk"`
	CreateAt     time.Time       `db:"notNull"`
	Name         string          `db:"index=name_alias"`
	Alias        string          `db:"index=name_alias"`
	Age          int             `db:"index"`
	IdentityCard string          `db:"uk"`
	IgnoreColumn string          `db:"-"`
	JsonColumn   json.RawMessage `json:"jsonColumn"`
	JsonMap      json.RawMessage `json:"jsonMap" db:"index"`
	Address      netip.Addr      `json:"address"`
}

type EmbedModel struct {
	Id           int64     `db:"pk"`
	CreateAt     time.Time `db:"notNull"`
	Name         string    `db:"index=name_alias"`
	Alias        string    `db:"index=name_alias"`
	Age          int       `db:"index"`
	IdentityCard string    `db:"uk"`
	IgnoreColumn string    `db:"-"`
	Child
	IgnoreChild Child `db:"-"`
}

type Child struct {
	Friends []string `json:"friends"`
	Email   string   `json:"email"`
}

type DuplicateModel struct {
	Id       int64     `db:"pk"`
	CreateAt time.Time `db:"notNull"`
	Name     string    `db:"index=name_alias"`
	Alias    string    `db:"index=name_alias"`
	Email    string    `db:"index"`
	Child
}

func TestDB_genCreateTableSql(t *testing.T) {
	var datas = []struct {
		name          string
		model         any
		expectErr     error
		expectSqlList []string
	}{
		{
			name:      "simple-table",
			model:     Model{},
			expectErr: nil,
			expectSqlList: []string{`CREATE TABLE IF NOT EXISTS model (
id BIGINT,
create_at TIMESTAMP,
name TEXT,
age INTEGER,
address CIDR
);`},
		},
		{
			name:      "index-table",
			model:     IndexModel{},
			expectErr: nil,
			expectSqlList: []string{`CREATE TABLE IF NOT EXISTS index_model (
id BIGINT PRIMARY KEY,
create_at TIMESTAMP NOT NULL,
name TEXT,
alias TEXT,
age INTEGER,
identity_card TEXT UNIQUE,
json_column JSONB,
json_map JSONB,
address CIDR
);`,
				`CREATE INDEX IF NOT EXISTS idx_index_model_age ON index_model (age);`,
				`CREATE INDEX IF NOT EXISTS idx_index_model_json_map ON index_model USING GIN (json_map);`,
				`CREATE INDEX IF NOT EXISTS idx_index_model_name_alias ON index_model (name, alias);`,
			},
		},
		{
			name:      "embed-table",
			model:     EmbedModel{},
			expectErr: nil,
			expectSqlList: []string{`CREATE TABLE IF NOT EXISTS embed_model (
id BIGINT PRIMARY KEY,
create_at TIMESTAMP NOT NULL,
name TEXT,
alias TEXT,
age INTEGER,
identity_card TEXT UNIQUE,
friends TEXT[],
email TEXT
);`,
				`CREATE INDEX IF NOT EXISTS idx_embed_model_age ON embed_model (age);`,
				`CREATE INDEX IF NOT EXISTS idx_embed_model_name_alias ON embed_model (name, alias);`,
			},
		},
		{
			name:          "duplicate-table",
			model:         DuplicateModel{},
			expectErr:     fmt.Errorf("column email already exists"),
			expectSqlList: nil,
		},
	}

	db := initDB(nil)
	for _, data := range datas {
		t.Run(data.name, func(t *testing.T) {
			sqlList, err := db.genCreateTableSql(data.model)
			for _, sql := range sqlList {
				t.Logf("sql: %+v", sql)
			}
			require.Equal(t, data.expectErr, err)
			require.Equal(t, data.expectSqlList, sqlList)
		})
	}
}

func TestDB_RegisterModels(t *testing.T) {
	db, err := NewDB(ConnStr)
	require.NoError(t, err)
	assert.NoError(t, db.RegisterModels(Model{}, IndexModel{}, EmbedModel{}))
}
