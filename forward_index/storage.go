package forward_index

import "fmt"

var storageOpenFunction = map[string]func(path string) (Storage, error){
	"badger":  OpenBadger,
	"bolt":    OpenBolt,
	"rocksdb": OpenRocksdb,
}

type Storage interface {
	GetPath() string
	Set(k, v []byte) error
	BatchSet(keys, values [][]byte) error
	Get(k []byte) ([]byte, error)
	BatchGet(keys [][]byte) ([][]byte, error) //BatchGet 不保证顺序
	Delete(k []byte) error
	BatchDelete(keys [][]byte) error
	Has(k []byte) bool
	IterDB(fn func(k, v []byte) error) int64
	IterKey(fn func(k []byte) error) int64
	Close() error
}

func OpenStorage(storeName string, path string) (Storage, error) {
	if fc, exists := storageOpenFunction[storeName]; exists {
		return fc(path)
	} else {
		return nil, fmt.Errorf("unsupported storage engine: %v", storeName)
	}
}
