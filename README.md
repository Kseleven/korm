# korm 
a Golang ORM library for postgres and pgx.

## Feature
* Simple data mapping with struct tags
* Explicit by design, no magic or conventions
* Insert database records from an annotated struct
* Select database records into an annotated struct or slice
* Compatible with pgx Exec/Query interface i.e. pgxpool.Pool, pgx.Conn, pgx.Tx
* Support for default ("auto generated") values, transient fields and all pgx types e.g. jsonb maps and slices

## Install
```shell
go get github.com/Kseleven/korm
```

## Usage
//TODO