package lessgo

import (
	"fmt"
)

// 一旦注册，不可再更改
type MiddlewareObj struct {
	Name        string // 全局唯一
	Description string
	Middleware
}

// 全局中间件登记
var middlewareMap = map[string]MiddlewareObj{}

// 必须在init()中调用
func RegMiddleware(name, description string, middleware interface{}) error {
	if _, ok := middlewareMap[name]; ok {
		err := fmt.Errorf("RegisterMiddleware called twice for middleware %v.", name)
		DefLessgo.Logger().Error("%v", err)
		return err
	}
	middlewareMap[name] = MiddlewareObj{
		Name:        name,
		Description: description,
		Middleware:  WrapMiddleware(middleware),
	}
	return nil
}

func MiddlewareMap() map[string]MiddlewareObj {
	return middlewareMap
}

func existMiddleware(name string) bool {
	_, ok := middlewareMap[name]
	return ok
}

func middlewareExistCheck(node *DynaRouter) error {
	var errstring string
	for _, m := range node.Middlewares {
		if !existMiddleware(m) {
			errstring += " \"" + m + "\""
		}
	}
	if len(errstring) == 0 {
		return nil
	}
	return fmt.Errorf("Specified below middlewares does not exist: %v", errstring)
}

func getMiddlewares(names []string) []Middleware {
	mws := make([]Middleware, len(names))
	for i, mw := range names {
		mws[i] = middlewareMap[mw]
	}
	return mws
}
