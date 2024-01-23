package kvdb

import (
	"os"
	"strings"

	"github.com/Orisun/radic/v2/util"
)

// 几种常见的基于LSM-tree算法实现的KV数据库
const (
	BOLT = iota
	BADGER
)

// redis也是一种KV数据库，读者可以自行用redis实现IKeyValueDB接口
type IKeyValueDB interface {
	Open() error                              //初始化DB
	GetDbPath() string                        //获取存储数据的目录
	Set(k, v []byte) error                    //写入<key, value>
	BatchSet(keys, values [][]byte) error     //批量写入<key, value>
	Get(k []byte) ([]byte, error)             //读取key对应的value
	BatchGet(keys [][]byte) ([][]byte, error) //批量读取，注意不保证顺序
	Delete(k []byte) error                    //删除
	BatchDelete(keys [][]byte) error          //批量删除
	Has(k []byte) bool                        //判断某个key是否存在
	IterDB(fn func(k, v []byte) error) int64  //遍历数据库，返回数据的条数
	IterKey(fn func(k []byte) error) int64    //遍历所有key，返回数据的条数
	Close() error                             //把内存中的数据flush到磁盘，同时释放文件锁
}

// Factory工厂模式，把类的创建和使用分隔开。Get函数就是一个工厂，它返回产品的接口，即它可以返回各种各样的具体产品。
func GetKvDb(dbtype int, DataDir string) (IKeyValueDB, error) { //通过Get函数【使用类】
	dirPath := strings.Split(DataDir, "/")
	baseDir := strings.Join(dirPath[0:len(dirPath)-1], "/")

	info, err := os.Stat(baseDir)
	if os.IsNotExist(err) {
		util.Log.Printf("create dir %s", baseDir)
		os.MkdirAll(baseDir, 0o600)
	} else {
		if info.Mode().IsRegular() {
			util.Log.Printf("%s is a regular file, will delete it", baseDir)
			os.Remove(baseDir)
		}
	}

	var db IKeyValueDB
	switch dbtype {
	case BADGER:
		db = new(Badger).WithDataPath(DataDir)
	default: //默认使用bolt
		db = new(Bolt).WithDataPath(DataDir).WithBucket("radic") //Builder生成器模式
	}
	err = db.Open() //创建具体KVDB的细节隐藏在Open()函数里。在这里【创建类】
	return db, err
}
