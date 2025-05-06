package korm

type Driver interface {
	Close() error
	Begin() (Transaction, error)
	GetDBPattern() DBPattern
	GetTableCache(name string) (*Field, bool)
	Exec(sql string, args ...any) (int64, error)
}

type Transaction interface {
	Insert(data any) error
	Select(target any, query string, args ...any) error
	Exec(query string, args ...any) (int64, error)
	Rollback() error
	Commit() error
}

func WithTx(d Driver, f func(tx Transaction) error) error {
	tx, err := d.Begin()
	if err != nil {
		return err
	}

	if err = f(tx); err != nil {
		tx.Rollback()
		return err
	}
	return tx.Commit()
}
