package korm

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestReadEnv(t *testing.T) {
	data, err := readEnv()
	require.NoError(t, err)
	t.Log(data)
}

func readEnv() (string, error) {
	data, err := os.ReadFile(".env")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func TestNewDB(t *testing.T) {
	connStr, err := readEnv()
	require.NoError(t, err)
	db, err := NewDB(connStr)
	require.NoError(t, err)
	defer db.Close()
}
