// Date         : 2018/7/28
// Author       : zhangchaoyang
// Description  :
package storage

import (
	"errors"
	"os"
	"path"
	"sync/atomic"

	"github.com/Orisun/radic/util"
	"github.com/dgraph-io/badger"
	"github.com/dgraph-io/badger/options"
)

type Badger struct {
	db   *badger.DB
	path string
}

var badgerOptions = badger.Options{
	LevelOneSize:        64 << 20, //第一层大小
	LevelSizeMultiplier: 10,       //下一层是上一层的多少倍
	MaxLevels:           12,       //LSM tree最多几层
	//key存在内存中，values(实际上value指针)存在磁盘中--称为vlog file
	TableLoadingMode:        options.MemoryMap, //LSM tree完全载入内存。MemoryMap是省内存，但查询速度比LoadToRAM慢一倍
	ValueLogLoadingMode:     options.FileIO,    //使用FileIO而非MemoryMap可以节省大量内存
	MaxTableSize:            8 << 20,           //8M
	NumCompactors:           8,                 //compaction线程数
	NumLevelZeroTables:      4,
	NumLevelZeroTablesStall: 10,
	NumMemtables:            4,     //写操作立即反应在MemTable上，当MemTable达到一定的大小时，它被刷新到磁盘，作为一个不可变的SSTable
	SyncWrites:              false, //异步写磁盘。即实时地去写内存中的LSM tree，当数据量达到MaxTableSize时，才对数据进行compaction然后写入磁盘。当调用Close时也会把内存中的数据flush到磁盘
	NumVersionsToKeep:       1,
	ValueLogFileSize:        64 << 20, //单位：字节。vlog文件超过这么大时就分裂文件。64M
	ValueLogMaxEntries:      100000,
	ValueThreshold:          32,
}

func OpenBadger(dbPath string) (Storage, error) {
	if err := os.MkdirAll(path.Dir(dbPath), os.ModePerm); err != nil { //如果dbPath对应的文件夹已存在则什么都不做，如果dbPath对应的文件已存在则返回错误
		return nil, err
	}

	badgerOptions.Dir = dbPath
	badgerOptions.ValueDir = dbPath
	db, err := badger.Open(badgerOptions) //文件只能被一个进程使用，如果不调用Close则下次无法Open。手动释放锁的办法：把LOCK文件删掉
	if err != nil {
		panic(err)
	}

	return &Badger{db, dbPath}, err
}

func (s *Badger) GetPath() string {
	return s.path
}

func (s *Badger) CheckAndGC() {
	lsmSize1, vlogSize1 := s.db.Size()
	for {
		if err := s.db.RunValueLogGC(0.5); err == badger.ErrNoRewrite || err == badger.ErrRejected {
			break
		}
	}
	lsmSize2, vlogSize2 := s.db.Size()
	if vlogSize2 < vlogSize1 {
		util.Log.Printf("badger before GC, LSM %d, vlog %d. after GC, LSM %d, vlog %d", lsmSize1, vlogSize1, lsmSize2, vlogSize2)
	} else {
		util.Log.Printf("collect zero garbage")
	}
}

// Set 为单个写操作开一个事务
func (s *Badger) Set(k, v []byte) error {
	err := s.db.Update(func(txn *badger.Txn) error { //db.Update相当于打开了一个读写事务:db.NewTransaction(true)。用db.Update的好处在于不用显式调用Txn.Commit()了
		//duration := time.Hour * 87600
		return txn.Set(k, v) //duration是能存活的时长
	})
	return err
}

// BatchSet 多个写操作使用一个事务
func (s *Badger) BatchSet(keys, values [][]byte) error {
	if len(keys) != len(values) {
		return errors.New("key value not the same length")
	}
	var err error
	txn := s.db.NewTransaction(true)
	for i, key := range keys {
		value := values[i]
		//duration := time.Hour * 87600
		//util.util.Log.Debugf("duration",duration)
		if err = txn.Set(key, value); err != nil {
			_ = txn.Commit() //发生异常时就提交老事务，然后开一个新事务，重试set
			txn = s.db.NewTransaction(true)
			_ = txn.Set(key, value)
		}
	}
	txn.Commit()
	return err
}

// Get 如果key不存在会返回error:Key not found
func (s *Badger) Get(k []byte) ([]byte, error) {
	var ival []byte
	err := s.db.View(func(txn *badger.Txn) error { //db.View相当于打开了一个读写事务:db.NewTransaction(true)。用db.Update的好处在于不用显式调用Txn.Discard()了
		item, err := txn.Get(k)
		if err != nil {
			return err
		}
		//buffer := make([]byte, badgerOptions.ValueLogMaxEntries)
		//ival, err = item.ValueCopy(buffer) //item只能在事务内部使用，如果要在事务外部使用需要通过ValueCopy
		err = item.Value(func(val []byte) error {
			ival = val
			return nil
		})
		return err
	})
	return ival, err
}

// BatchGet 返回的values与传入的keys顺序保持一致。如果key不存在或读取失败则对应的value是空数组
func (s *Badger) BatchGet(keys [][]byte) ([][]byte, error) {
	var err error
	txn := s.db.NewTransaction(false) //只读事务
	values := make([][]byte, len(keys))
	for i, key := range keys {
		var item *badger.Item
		item, err = txn.Get(key)
		if err == nil {
			//buffer := make([]byte, badgerOptions.ValueLogMaxEntries)
			var ival []byte
			//ival, err = item.ValueCopy(buffer)
			err = item.Value(func(val []byte) error {
				ival = val
				return nil
			})
			if err == nil {
				values[i] = ival
			} else { //拷贝失败
				values[i] = []byte{} //拷贝失败就把value设为空数组
			}
		} else { //读取失败
			values[i] = []byte{}              //读取失败就把value设为空数组
			if err != badger.ErrKeyNotFound { //如果真的发生异常，则开一个新事务继续读后面的key
				txn.Discard()
				txn = s.db.NewTransaction(false)
			}
		}
	}
	txn.Discard() //只读事务调Discard就可以了，不需要调Commit。Commit内部也会调Discard
	return values, err
}

// Delete
func (s *Badger) Delete(k []byte) error {
	err := s.db.Update(func(txn *badger.Txn) error {
		return txn.Delete(k)
	})
	return err
}

// BatchDelete
func (s *Badger) BatchDelete(keys [][]byte) error {
	var err error
	txn := s.db.NewTransaction(true)
	for _, key := range keys {
		if err = txn.Delete(key); err != nil {
			_ = txn.Commit() //发生异常时就提交老事务，然后开一个新事务，重试delete
			txn = s.db.NewTransaction(true)
			_ = txn.Delete(key)
		}
	}
	txn.Commit()
	return err
}

// Has 判断某个key是否存在
func (s *Badger) Has(k []byte) bool {
	var exists = false
	s.db.View(func(txn *badger.Txn) error { //db.View相当于打开了一个读写事务:db.NewTransaction(true)。用db.Update的好处在于不用显式调用Txn.Discard()了
		_, err := txn.Get(k)
		if err != nil {
			return err
		} else {
			exists = true //没有任何异常发生，则认为k存在。如果k不存在会发生ErrKeyNotFound
		}
		return err
	})
	return exists
}

// IterDB 遍历整个DB
func (s *Badger) IterDB(fn func(k, v []byte) error) int64 {
	var total int64
	s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			key := item.Key()

			var ival []byte
			//var err error
			//buffer := make([]byte, badgerOptions.ValueLogMaxEntries)
			//ival, err = item.ValueCopy(buffer)

			err := item.Value(func(val []byte) error {
				ival = val
				return nil
			})

			if err != nil {
				continue
			}
			if err := fn(key, ival); err == nil {
				atomic.AddInt64(&total, 1)
			}
		}
		return nil
	})
	return atomic.LoadInt64(&total)
}

// IterKey 只遍历key。key是全部存在LSM tree上的，只需要读内存，所以很快
func (s *Badger) IterKey(fn func(k []byte) error) int64 {
	var total int64
	s.db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false //只需要读key，所以把PrefetchValues设为false
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			k := item.Key()
			if err := fn(k); err == nil {
				atomic.AddInt64(&total, 1)
			}
		}
		return nil
	})
	return atomic.LoadInt64(&total)
}

func (s *Badger) Size() (int64, int64) {
	return s.db.Size()
}

// Close 把内存中的数据flush到磁盘，同时释放文件锁。如果没有close，再open时会丢失很多数据
func (s *Badger) Close() error {
	return s.db.Close()
}
