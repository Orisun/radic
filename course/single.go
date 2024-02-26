package course

import (
	"fmt"
	"sync"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var single *gorm.DB //通过gorm.Open()创建的gorm.DB是一个连接池，只需要创建它的一个实例
var once sync.Once = sync.Once{}
var lock = &sync.Mutex{}

func GetDB1() *gorm.DB {
	if single == nil { //先判断是否为nil，避免无谓的上锁
		lock.Lock()
		defer lock.Unlock()
		if single == nil { //需要二次确认实例尚未创建。如果用sync.Once就不用二次确认
			single, _ = gorm.Open(mysql.Open(""))
		} else {
			fmt.Println("单例已经创建过了")
		}
	} else {
		fmt.Println("单例已经创建过了")
	}

	return single
}

// func init() { //init()只会执行一次，所以可以实现单例。但使用init()通常要小心代码的各种依赖关系，关心代码的执行顺序
// 	single, _ = gorm.Open(mysql.Open(""))
// }

// func GetDB2() *gorm.DB {
// 	return single
// }

func GetDB3() *gorm.DB {
	if single == nil { //先判断是否为nil，避免无谓的once.Do
		once.Do(
			func() {
				single, _ = gorm.Open(mysql.Open(""))
			})
	} else {
		fmt.Println("单例已经创建过了")
	}

	return single
}
