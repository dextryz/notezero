package badger

import (
	"github.com/dgraph-io/badger/v4"
)

type Cache struct {
	*badger.DB
}

func New(db *badger.DB) (*Cache, error) {
	return &Cache{
		DB: db,
	}, nil
}

func (c *Cache) Get(key string) ([]byte, bool) {
	var val []byte
	err := c.View(func(txn *badger.Txn) error {
		b, err := txn.Get([]byte(key))
		if err != nil {
			return err
		}

		val, err = b.ValueCopy(nil)
		return err
	})

	if err == badger.ErrKeyNotFound {
		return nil, false
	}
	if err != nil {
		panic(err)
	}

	return val, true
}

func (c *Cache) Set(key string, value []byte) {
	err := c.Update(func(txn *badger.Txn) error {
		return txn.Set([]byte(key), value)
	})
	if err != nil {
		panic(err)
	}
}

func (c *Cache) Delete(key string) error {
	return c.Update(func(txn *badger.Txn) error {
		return txn.Delete([]byte(key))
	})
}
