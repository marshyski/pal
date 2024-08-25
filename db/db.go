package db

import (
	"fmt"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/perlogix/pal/config"
)

// indexCacheSize = 100MB
const indexCacheSize = 100 << 20

type DB struct {
	badgerDB *badger.DB
}

func Open() (*DB, error) {
	badgerDB, err := badger.Open(
		badger.
			DefaultOptions(config.GetConfigStr("store_db_path")).
			WithCompression(options.ZSTD).
			WithZSTDCompressionLevel(1).
			WithEncryptionKey([]byte(config.GetConfigStr("store_encrypt_key"))).
			WithIndexCacheSize(indexCacheSize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	return &DB{
		badgerDB: badgerDB,
	}, nil
}

func (s *DB) Close() error {
	if err := s.badgerDB.Close(); err != nil {
		return fmt.Errorf("failed to close badger db: %w", err)
	}

	return nil
}

func (s *DB) Get(key string) (string, error) {
	var val []byte

	err := s.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to get state from key: %s - %w", key, err)
		}

		val, err = item.ValueCopy(nil)
		if err != nil {
			return fmt.Errorf("failed to copy value from key: %s - %w", key, err)
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("failed to get state from key: %s - %w", key, err)
	}

	if len(val) == 0 {
		return "", nil
	}

	return string(val), nil
}

func (s *DB) Put(key string, val string) error {

	err := s.badgerDB.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(key), []byte(val))
		if err != nil {
			return fmt.Errorf("failed to set state for key: %s - %w", key, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update state for key: %s - %w", key, err)
	}

	return nil
}

func (s *DB) Delete(key string) error {
	err := s.badgerDB.Update(func(txn *badger.Txn) error {
		err := txn.Delete([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to delete state for key: %s - %w", key, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to update state for key: %s - %w", key, err)
	}

	return nil
}
