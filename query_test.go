package korm

import (
	"encoding/json"
	"fmt"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestQuery(t *testing.T) {
	connStr, err := readEnv()
	require.NoError(t, err)
	db, err := NewDB(connStr)
	require.NoError(t, err)
	require.NoError(t, db.RegisterModels(IndexModel{}, EmbedModel{}))
	num := 10
	indexModels := make([]*IndexModel, num)

	addr := netip.MustParseAddr("192.168.1.0")
	jsData := `{"test": "test", "num": %d}`
	for i := 0; i < num; i++ {
		addr = addr.Next()
		idx := &IndexModel{
			Id:           int64(i),
			CreateAt:     time.Now(),
			Name:         fmt.Sprintf("name%d", i),
			Alias:        fmt.Sprintf("alias%d", i),
			Age:          i,
			IdentityCard: fmt.Sprintf("card%d", i),
			IgnoreColumn: fmt.Sprintf("ignore column%d", i),
			JsonColumn:   json.RawMessage([]byte(fmt.Sprintf(jsData, i))),
			JsonMap:      json.RawMessage([]byte(fmt.Sprintf(jsData, i))),
			Address:      addr,
		}
		indexModels[i] = idx
	}

	var result []*IndexModel
	require.NoError(t, WithTx(db, func(tx Transaction) error {
		if _, err := tx.Exec("DELETE FROM index_model"); err != nil {
			return fmt.Errorf("delete index_model failed: %w", err)
		}
		if err := tx.Insert(indexModels); err != nil {
			return err
		}

		if err := tx.Select(&result, "SELECT * FROM index_model ORDER BY id"); err != nil {
			return fmt.Errorf("query failed: %w", err)
		}
		return nil
	}))

	t.Logf("result len: %d\n", len(result))
	for _, model := range result {
		t.Logf("model: %+v\n", model)
	}
}

func TestQueryEmbed(t *testing.T) {
	connStr, err := readEnv()
	require.NoError(t, err)
	db, err := NewDB(connStr)
	require.NoError(t, err)
	require.NoError(t, db.RegisterModels(EmbedModel{}))
	num := 10
	embedModels := make([]*EmbedModel, num)

	addr := netip.MustParseAddr("192.168.1.0")
	for i := 0; i < num; i++ {
		addr = addr.Next()
		idx := &EmbedModel{
			Id:           int64(i),
			Name:         fmt.Sprintf("name%d", i),
			Alias:        fmt.Sprintf("alias%d", i),
			Age:          i,
			IdentityCard: fmt.Sprintf("card%d", i),
			IgnoreColumn: fmt.Sprintf("ignore column%d", i),
			IgnoreChild: Child{
				Friends: []string{"1", "2"},
				Email:   fmt.Sprintf("ignore%d", i),
			},
			Child: Child{
				Friends: []string{"3", "4"},
				Email:   fmt.Sprintf("email%d", i),
			},
		}
		embedModels[i] = idx
	}

	var result []*EmbedModel
	require.NoError(t, WithTx(db, func(tx Transaction) error {
		if _, err := tx.Exec("DELETE FROM embed_model"); err != nil {
			return fmt.Errorf("delete embed_model failed: %w", err)
		}
		if err := tx.Insert(embedModels); err != nil {
			return err
		}

		if err := tx.Select(&result, "SELECT * FROM embed_model ORDER BY id"); err != nil {
			return fmt.Errorf("query 1 failed: %w", err)
		}
		return nil
	}))

	t.Logf("result len: %d\n", len(result))
	for _, model := range result {
		t.Logf("model: %+v\n", model)
	}
}

func BenchmarkQueryEmbed(b *testing.B) {
	connStr, err := readEnv()
	require.NoError(b, err)
	db, err := NewDB(connStr)
	require.NoError(b, err)
	require.NoError(b, db.RegisterModels(EmbedModel{}))
	num := 10
	embedModels := make([]*EmbedModel, num)

	addr := netip.MustParseAddr("192.168.1.0")
	for i := 0; i < num; i++ {
		addr = addr.Next()
		idx := &EmbedModel{
			Id:           int64(i),
			Name:         fmt.Sprintf("name%d", i),
			Alias:        fmt.Sprintf("alias%d", i),
			Age:          i,
			IdentityCard: fmt.Sprintf("card%d", i),
			IgnoreColumn: fmt.Sprintf("ignore column%d", i),
			IgnoreChild: Child{
				Friends: []string{"1", "2"},
				Email:   fmt.Sprintf("ignore%d", i),
			},
			Child: Child{
				Friends: []string{"3", "4"},
				Email:   fmt.Sprintf("email%d", i),
			},
		}
		embedModels[i] = idx
	}

	//var result []EmbedModel
	var result []*EmbedModel
	require.NoError(b, WithTx(db, func(tx Transaction) error {
		if _, err := tx.Exec("DELETE FROM embed_model"); err != nil {
			return fmt.Errorf("delete embed_model failed: %w", err)
		}
		if err := tx.Insert(embedModels); err != nil {
			return err
		}
		b.ResetTimer()

		//BenchmarkQueryEmbed-12    	    3265	    343254 ns/op	   11850 B/op	     340 allocs/op
		//BenchmarkQueryEmbed-12    	    3327	    313296 ns/op	   11842 B/op	     340 allocs/op
		//BenchmarkQueryEmbed-12    	    3628	    351936 ns/op	   11789 B/op	     340 allocs/op
		for i := 0; i < b.N; i++ {
			if err := tx.Select(&result, "SELECT * FROM embed_model ORDER BY id"); err != nil {
				return fmt.Errorf("query 1 failed: %w", err)
			}
		}

		//BenchmarkQueryEmbed-12    	    3459	    297968 ns/op	   12946 B/op	     291 allocs/op
		//BenchmarkQueryEmbed-12    	    4582	    333441 ns/op	   12947 B/op	     291 allocs/op
		//BenchmarkQueryEmbed-12    	    3471	    296097 ns/op	   12946 B/op	     291 allocs/op
		//for i := 0; i < b.N; i++ {
		//	if result, err = QueryG[EmbedModel](db, tx, "SELECT * FROM embed_model ORDER BY id"); err != nil {
		//		return fmt.Errorf("query failed: %w", err)
		//	}
		//}
		return nil
	}))
	b.Log(len(result))
}
