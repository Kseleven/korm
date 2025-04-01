package korm

type QueryInter interface {
	GenQuery() (string, []any)
}

type Updater interface{}
