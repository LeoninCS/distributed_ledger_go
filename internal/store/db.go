package store

import "github.com/dgraph-io/badger/v3"

type Store struct {
	db *badger.DB
}

func NewStore(db *badger.DB) *Store {
	return &Store{
		db: db,
	}
}
