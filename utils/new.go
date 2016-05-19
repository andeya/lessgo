package utils

import (
	"reflect"
)

// 复制一个对象，返回指针类型的空对象
func NewObjectPtr(i interface{}) interface{} {
	t := reflect.TypeOf(i)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return reflect.New(t).Interface()
}
