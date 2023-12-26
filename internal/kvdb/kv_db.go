package kvdb

//几种常见的基于LSM-tree算法实现的KV数据库
const (
	BOLT = iota
	BADGER
)

// Stratergy模式，完成同一个任务可以有多种不同的实现方案
//
// redis也是一种KV数据库，读者可以自行用redis实现IKeyValueDB接口
type IKeyValueDB interface {
	Open(DataDir string) error                //初始化DB
	GetDbPath() string                        //获取存储数据的目录
	Set(k, v []byte) error                    //写入<key, value>
	BatchSet(keys, values [][]byte) error     //批量写入<key, value>
	Get(k []byte) ([]byte, error)             //读取key对应的value
	BatchGet(keys [][]byte) ([][]byte, error) //批量读取，注意不保证顺序
	Delete(k []byte) error                    //删除
	BatchDelete(keys [][]byte) error          //批量删除
	Has(k []byte) bool                        //判断某个key是否存在
	IterDB(fn func(k, v []byte) error) int64  //遍历数据库
	IterKey(fn func(k []byte) error) int64    //遍历所有key
	Close() error                             //把内存中的数据flush到磁盘，同时释放文件锁
}

// Factory模式，把类的创建和使用分隔开。Get函数就是一个工厂，它返回产品的接口，即它可以返回各种各样的具体产品。
func GetKvDb(dbtype int, DataDir string) (IKeyValueDB, error) { //通过Get函数【使用类】
	var db IKeyValueDB
	var err error
	switch dbtype {
	case BADGER:
		db = new(Badger)
	default: //默认使用bolt
		db = new(Bolt).WithBucket("radic") //Builder模式
	}
	err = db.Open(DataDir) //创建具体KVDB的细节隐藏在Open()函数里。在这里【创建类】
	return db, err
}
