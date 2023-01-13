// Date         : 2019-02-05 22:30
// Author       : zhangzhaoyang
// Description  :

package storage

import (
	"errors"
	bolt "go.etcd.io/bbolt"
	"sync/atomic"
)

var (
	gdocs       = []byte("gdocs")
	boltOptions = bolt.DefaultOptions
)

// Bolt bolt store struct
type Bolt struct {
	db   *bolt.DB
	path string
}

// OpenBolt open Bolt store
func OpenBolt(dbPath string) (Storage, error) {
	db, err := bolt.Open(dbPath, 0600, boltOptions)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(gdocs)
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Bolt{db: db, path: dbPath}, err
}

func (s *Bolt) GetPath() string {
	return s.path
}

// WALName returns the path to currently open database file.
func (s *Bolt) WALName() string {
	return s.db.Path()
}

// Set executes a function within the context of a read-write managed
// transaction. If no error is returned from the function then the transaction
// is committed. If an error is returned then the entire transaction is rolled back.
// Any error that is returned from the function or returned from the commit is returned
// from the Update() method.
func (s *Bolt) Set(k, v []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(gdocs).Put(k, v)
	})
}

func (s *Bolt) BatchSet(keys, values [][]byte) error {
	if len(keys) != len(values) {
		return errors.New("key value not the same length")
	}
	var err error
	s.db.Batch(func(tx *bolt.Tx) error {
		for i, key := range keys {
			value := values[i]
			tx.Bucket(gdocs).Put(key, value)
		}
		return nil
	})
	return err
}

// Get executes a function within the context of a managed read-only transaction.
// Any error that is returned from the function is returned from the View() method.
func (s *Bolt) Get(k []byte) ([]byte, error) {
	var ival []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		ival = tx.Bucket(gdocs).Get(k)
		return nil
	})
	return ival, err
}

func (s *Bolt) BatchGet(keys [][]byte) ([][]byte, error) {
	var err error
	values := make([][]byte, len(keys))
	s.db.Batch(func(tx *bolt.Tx) error {
		for i, key := range keys {
			ival := tx.Bucket(gdocs).Get(key)
			values[i] = ival
		}
		return nil
	})
	return values, err
}

// Delete deletes a key. Exposing this so that user does not
// have to specify the Entry directly.
func (s *Bolt) Delete(k []byte) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		return tx.Bucket(gdocs).Delete(k)
	})
}

func (s *Bolt) BatchDelete(keys [][]byte) error {
	var err error
	s.db.Batch(func(tx *bolt.Tx) error {
		for _, key := range keys {
			tx.Bucket(gdocs).Delete(key)
		}
		return nil
	})
	return err
}

// Has returns true if the DB does contains the given key.
func (s *Bolt) Has(k []byte) bool {
	// return s.db.Exists(k)
	var b []byte
	err := s.db.View(func(tx *bolt.Tx) error {
		b = tx.Bucket(gdocs).Get(k)
		return nil
	})

	// b == nil
	if err != nil || string(b) == "" {
		return false
	}

	return true
}

// ForEach get all key and value
func (s *Bolt) IterDB(fn func(k, v []byte) error) int64 {
	var total int64
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(gdocs)
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			if err := fn(k, v); err != nil {
				return err
			} else {
				atomic.AddInt64(&total, 1)
			}
		}
		return nil
	})

	return atomic.LoadInt64(&total)
}

func (s *Bolt) IterKey(fn func(k []byte) error) int64 {
	var total int64
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(gdocs)
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if err := fn(k); err != nil {
				return err
			} else {
				atomic.AddInt64(&total, 1)
			}
		}
		return nil
	})
	return atomic.LoadInt64(&total)
}

func (s *Bolt) Size() (int64, int64) {
	return s.Size()
}

// Close releases all database resources. All transactions
// must be closed before closing the database.
func (s *Bolt) Close() error {
	return s.db.Close()
}
