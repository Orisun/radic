package course

import (
	"fmt"
	"reflect"
)

func main() {

}

type IError interface {
	Error() string
}

type ErrA struct {
	Code int
}

func (ErrA) Error() string {
	return "龙年"
}

type ErrB struct{}

func (ErrB) Error() string {
	return "行大运"
}

func ReflectError(err IError) {
	tp := reflect.TypeOf(err)            //先观察一下err是什么类型
	fmt.Println(tp, tp.Elem().PkgPath()) //指针类型需要先通过Elem()转为非指针类型，才能获得PkgPath
	if errA, ok := err.(*ErrA); ok {
		if errA.Code == 1062 {
			fmt.Println(errA.Error())
		}
	}
}
