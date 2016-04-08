package lessgo

import (
	"fmt"
)

// 一旦注册，不可再更改
type MiddlewareObj struct {
	Name        string // 全局唯一
	Description string
	MiddlewareFunc
}

// 登记全局中间件
var middlewareMap = map[string]MiddlewareObj{}

func init() {
	RegMiddleware("检查网站是否开启", "", checkServer())
	RegMiddleware("自动匹配home页面", "", checkHome())
	RegMiddleware("运行时请求日志", "", RequestLogger())
	RegMiddleware("异常恢复", "", Recover())
}

// 必须在init()中调用
func RegMiddleware(name, description string, middleware interface{}) error {
	if _, ok := middlewareMap[name]; ok {
		err := fmt.Errorf("RegisterMiddlewareFunc called twice for middleware %v.", name)
		DefLessgo.Logger().Error("%v", err)
		return err
	}
	middlewareMap[name] = MiddlewareObj{
		Name:           name,
		Description:    description,
		MiddlewareFunc: WrapMiddleware(middleware),
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

func middlewareExistCheck(node *VirtRouter) error {
	var errstring string
	for _, m := range node.AllMiddleware() {
		if !existMiddleware(m) {
			errstring += " \"" + m + "\""
		}
	}
	if len(errstring) == 0 {
		return nil
	}
	return fmt.Errorf("Specified below middlewares does not exist: %v\n", errstring)
}

func getMiddlewares(names []string) []MiddlewareFunc {
	mws := make([]MiddlewareFunc, len(names))
	for i, mw := range names {
		mws[i] = middlewareMap[mw].MiddlewareFunc
	}
	return mws
}
