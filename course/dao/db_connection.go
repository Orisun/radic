package dao

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/Orisun/radic/v2/util"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	ormlog "gorm.io/gorm/logger"
)

var (
	search_mysql      *gorm.DB
	search_mysql_once sync.Once
	dblog             ormlog.Interface
)

func init() {
	dblog = ormlog.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		ormlog.Config{
			SlowThreshold: 100 * time.Millisecond, // 慢 SQL 阈值
			LogLevel:      ormlog.Silent,          // Log level，Silent表示不输出日志
			Colorful:      false,                  // 禁用彩色打印
		},
	)
}

func createMysqlDB(dbname, host, user, pass string, port int) *gorm.DB {
	// data source name 是 tester:123456@tcp(localhost:3306)/blog?charset=utf8mb4&parseTime=True&loc=Local
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local", user, pass, host, port, dbname) //mb4兼容emoji表情符号
	var err error
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{Logger: dblog, PrepareStmt: true}) //启用PrepareStmt，SQL预编译，提高查询效率
	if err != nil {
		util.Log.Panicf("connect to mysql use dsn %s failed: %s", dsn, err) //panic() os.Exit(2)
	}
	//设置数据库连接池参数，提高并发性能
	sqlDB, _ := db.DB()
	sqlDB.SetMaxOpenConns(100) //设置数据库连接池最大连接数
	sqlDB.SetMaxIdleConns(20)  //连接池最大允许的空闲连接数，如果没有sql任务需要执行的连接数大于20，超过的连接会被连接池关闭。
	util.Log.Printf("connect to mysql db %s", dbname)
	return db
}

func GetSearchDBConnection() *gorm.DB { //单例
	if search_mysql == nil {
		search_mysql_once.Do(func() {
			search_mysql = createMysqlDB("search", "localhost", "tester", "123456", 3306)
		})
	}

	return search_mysql
}
