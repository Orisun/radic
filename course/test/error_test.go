package test

import (
	"fmt"
	"log"
	"reflect"
	"testing"

	"github.com/Orisun/radic/v2/course"
	"github.com/Orisun/radic/v2/course/dao"
	"github.com/go-sql-driver/mysql"
)

func TestReflect(t *testing.T) {
	db := dao.GetSearchDBConnection()
	video := dao.BiliVideo{
		Id:    "BV1xt4y1R7A8",
		Title: "6小时入门go语言",
	}
	err := db.Create(video).Error
	if err != nil {
		tp := reflect.TypeOf(err)                    //先观察一下err是什么类型。我们看到它是*mysql.MySQLError类型
		fmt.Println(tp, tp.Elem().PkgPath())         //指针类型需要先通过Elem()转为非指针类型，才能获得PkgPath
		if inst, ok := err.(*mysql.MySQLError); ok { //有了上面的观察，这里就有断言的方向了。当然有可能会断言失败
			if inst.Number != 1062 { //忽略1062这种错误
				log.Printf("写库失败：%s", err)
			}
		}
	}

}

func TestError(t *testing.T) {
	course.ReflectError(&course.ErrA{Code: 1062})
}
