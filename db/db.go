// SPDX-License-Identifier: AGPL-3.0-only
// pal - github.com/marshyski/pal
// Copyright (C) 2024-2025  github.com/marshyski

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package db

import (
	"errors"
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
	DBC = &DB{}
)

type DB struct {
	badgerDB *badger.DB
}

// getRestrictedKeys gets a constant string slice
func getRestrictedKeys() []string {
	return []string{"pal_notifications", "pal_groups"}
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

func (s *DB) Get(key string) (data.DBSet, error) {
	var dbSet data.DBSet

	err := s.badgerDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(key))
		if err != nil {
			return fmt.Errorf("failed to get state from key: %s - %w", key, err)
		}

		err = item.Value(func(val []byte) error {
			return json.Unmarshal(val, &dbSet)
		})
		if err != nil {
			return fmt.Errorf("failed to copy value from key: %s - %w", key, err)
		}

		return nil
	})

	if err != nil {
		return dbSet, fmt.Errorf("failed to get state from key: %s - %w", key, err)
	}

	return dbSet, nil
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
		return errors.New("failed to marshal JSON for key: pal_notifications")
	}

	err = s.badgerDB.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("pal_notifications"), jsonData)
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

func (s *DB) DeleteNotifications() error {
	jsonData, err := json.Marshal([]data.Notification{})
	if err != nil {
		return errors.New("failed to marshal JSON for key: pal_notifications")
	}

	err = s.badgerDB.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte("pal_notifications"), jsonData)
		if err != nil {
			return fmt.Errorf("failed to set state for key: pal_notifications - %w", err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete state for key: pal_notifications - %w", err)
	}

	return nil
}

func (s *DB) Put(dbSet data.DBSet) error {
	for _, e := range getRestrictedKeys() {
		if e == dbSet.Key {
			return fmt.Errorf("failed to add value to key %s due to restricted key denied", dbSet.Key)
		}
	}
	jsonData, err := json.Marshal(dbSet)
	if err != nil {
		return errors.New("failed to marshal JSON for key: " + dbSet.Key)
	}

	err = s.badgerDB.Update(func(txn *badger.Txn) error {
		err := txn.Set([]byte(dbSet.Key), jsonData)
		if err != nil {
			return fmt.Errorf("failed to set state for key: %s - %w", dbSet.Key, err)
		}

		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to update state for key: %s - %w", dbSet.Key, err)
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

func (s *DB) PutGroupAction(group string, action data.ActionData) {
	groups := DBC.GetGroupActions(group)

	for i, e := range groups {
		if e.Action == action.Action {
			groups[i] = action
		}
	}

	DBC.PutGroupActions(group, groups)
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

func (s *DB) Dump() []data.DBSet {
	var dbSetSlice []data.DBSet

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
				for _, e := range getRestrictedKeys() {
					if strings.HasPrefix(k, e) {
						restricted = true
					}
				}
				var dbSet data.DBSet
				if !restricted {
					err := json.Unmarshal(v, &dbSet)
					if err == nil {
						dbSetSlice = append(dbSetSlice, dbSet)
					}
				}
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})

	// TODO: ignoring err for now return empty map
	if err != nil {
		return dbSetSlice
	}

	return dbSetSlice
}
