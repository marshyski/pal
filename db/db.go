package db

import (
	"encoding/json"
	"fmt"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/marshyski/pal/config"
	"github.com/marshyski/pal/data"
)

// indexCacheSize = 100MB
const indexCacheSize = 100 << 20

var (
	DBC = &DB{}
)

type DB struct {
	badgerDB *badger.DB
}

func Open() (*DB, error) {
	badgerDB, err := badger.Open(
		badger.
			DefaultOptions(config.GetConfigStr("db_path")).
			WithCompression(options.ZSTD).
			WithZSTDCompressionLevel(1).
			WithEncryptionKey([]byte(config.GetConfigStr("db_encrypt_key"))).
			WithIndexCacheSize(indexCacheSize),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to open badger db: %w", err)
	}

	DBC = &DB{
		badgerDB: badgerDB,
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

func (s *DB) GetNotifications() []data.Notification {
	var retrievedData []data.Notification

	err := s.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("pal_notifications"))
		if err != nil {
			return fmt.Errorf("failed to get state from key: pal_notifications - %w", err)
		}

		err = item.Value(func(val []byte) error {
			err = json.Unmarshal(val, &retrievedData)
			if err != nil {
				return fmt.Errorf("failed to get unmarshal JSON data from key: pal_notifications - %w", err)
			}
			return nil
		})
		if err != nil {
			return fmt.Errorf("failed to copy value from key: pal_notifications - %w", err)
		}

		return nil
	})

	// skip error return empty obj
	if err != nil {
		return retrievedData
	}

	return retrievedData
}

func (s *DB) PutNotifications(notifications []data.Notification) error {
	jsonData, err := json.Marshal(notifications)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON for key: pal_notifications")
	}

	err = s.badgerDB.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("pal_notifications"), []byte(jsonData))
		if err != nil {
			return fmt.Errorf("failed to set state for key: pal_notifications - %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update state for key: pal_notifications - %w", err)
	}

	return nil
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

func (s *DB) Dump() map[string]string {
	keys := make(map[string]string)

	err := s.badgerDB.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchSize = 10
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := string(item.Key())
			err := item.Value(func(v []byte) error {
				if k != "pal_notifications" {
					keys[k] = string(v)
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	// ignoring err for now return empty map
	if err != nil {
		return keys
	}

	return keys
}
