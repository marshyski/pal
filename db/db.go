package db

import (
	"fmt"
	"log"
	"strings"

	badger "github.com/dgraph-io/badger/v4"
	"github.com/dgraph-io/badger/v4/options"
	"github.com/goccy/go-json"
	"github.com/marshyski/pal/config"
	"github.com/marshyski/pal/data"
)

// indexCacheSize = 100MB
const indexCacheSize = 100 << 20

var (
	DBC             = &DB{}
	restricted_keys = []string{"pal_notifications", "pal_groups"}
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

func (s *DB) GetNotifications(group string) []data.Notification {
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

	if group != "" {
		for i, e := range retrievedData {
			if e.Group != group {
				retrievedData = append(retrievedData[:i], retrievedData[i+1:]...)
			}
		}
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

func (s *DB) PutGroups(data map[string][]data.ActionData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to json.Marshal state for key: pal_groups - %w", err)
	}

	// Set the value in the database
	err = s.badgerDB.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("pal_groups"), jsonData)
		if err != nil {
			return fmt.Errorf("failed to set state for key: pal_groups - %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to set state for key: pal_groups - %w", err)
	}

	return nil
}

func (s *DB) GetGroups() map[string][]data.ActionData {
	var data = make(map[string][]data.ActionData)
	err := s.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("pal_groups"))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &data)
		})
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		// TODO: DEBUG STATEMENT
		log.Println(err.Error())
	}

	return data
}

func (s *DB) GetGroupActions(group string) []data.ActionData {
	var jsonData map[string][]data.ActionData
	var actions []data.ActionData
	err := s.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("pal_groups"))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			err := json.Unmarshal(val, &jsonData)
			if err != nil {
				return err
			}
			actions = jsonData[group]
			return nil
		})
		return err
	})
	if err != nil {
		return actions
	}
	return actions
}

func (s *DB) GetGroupAction(group, action string) data.ActionData {
	var jsonData map[string][]data.ActionData
	var actionData data.ActionData
	err := s.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte("pal_groups"))
		if err != nil {
			return err
		}

		err = item.Value(func(val []byte) error {
			err := json.Unmarshal(val, &jsonData)
			if err != nil {
				return err
			}
			return nil
		})
		return err
	})
	if err != nil {
		return actionData
	}

	for k, v := range jsonData {
		if k == group {
			for _, e := range v {
				if e.Action == action {
					return e
				}
			}
		}
	}

	return actionData
}

func (s *DB) PutGroupActions(group string, actions []data.ActionData) {
	groups := DBC.GetGroups()
	groups[group] = actions
	err := DBC.PutGroups(groups)
	if err != nil {
		// TODOD: DEBUG STATEMENT
		log.Println(err.Error())
	}
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
				var restricted bool
				for _, e := range restricted_keys {
					if strings.HasPrefix(k, e) {
						restricted = true
					}
				}
				if !restricted {
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
