package lessgo

import (
	"reflect"
	"runtime"
	"sort"

	"github.com/lessgo/lessgo/virtrouter"
)

var (
	// 全局虚拟路由
	DefVirtRouter, _ = virtrouter.NewVirtRouterRoot()
	// 全部handler及其id
	handlerMap = map[string]HandlerFunc{}
)

// 返回虚拟路由列表
func VirtRouterProgeny() []*virtrouter.VirtRouter {
	return DefVirtRouter.Progeny()
}

// 快速返回指定uri对应的虚拟路由节点
func GetVirtRouter(id string) (*virtrouter.VirtRouter, bool) {
	return virtrouter.GetVirtRouter(id)
}

// 必须在init()中调用
// 从根路由开始配置路由
func RootRouter(node ...*virtrouter.VirtRouter) {
	DefVirtRouter.AddChildren(node)
}

// 必须在init()中调用
// 配置路由分组
func SubRouter(prefix, name string, node ...*virtrouter.VirtRouter) *virtrouter.VirtRouter {
	return virtrouter.NewVirtRouterGroup(prefix, name).AddChildren(node)
}

// 必须在init()中调用
// 配置操作
func Get(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	return route([]string{GET}, prefix, name, descHandlerOrhandler, middleware)
}
func Head(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	return route([]string{HEAD}, prefix, name, descHandlerOrhandler, middleware)
}
func Options(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	return route([]string{OPTIONS}, prefix, name, descHandlerOrhandler, middleware)
}
func Patch(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	return route([]string{PATCH}, prefix, name, descHandlerOrhandler, middleware)
}
func Post(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	return route([]string{POST}, prefix, name, descHandlerOrhandler, middleware)
}
func Put(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	return route([]string{PUT}, prefix, name, descHandlerOrhandler, middleware)
}
func Trace(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	return route([]string{TRACE}, prefix, name, descHandlerOrhandler, middleware)
}
func Any(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	return route([]string{CONNECT, DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT, TRACE}, prefix, name, descHandlerOrhandler, middleware)
}
func Match(methods []string, prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *virtrouter.VirtRouter {
	if len(methods) == 0 {
		DefLessgo.logger.Error("The method can not be empty: %v", name)
	}
	return route(methods, prefix, name, descHandlerOrhandler, middleware)
}

func route(methods []string, prefix, name string, descHandlerOrhandler interface{}, middleware []string) *virtrouter.VirtRouter {
	sort.Strings(methods)
	hid := handleWareUri(descHandlerOrhandler, methods, prefix)

	var (
		handler                       HandlerFunc
		description, success, failure string
		param                         map[string]string
	)
	switch h := descHandlerOrhandler.(type) {
	case HandlerFunc:
		handler = h
	case func(Context) error:
		handler = HandlerFunc(h)
	case DescHandler:
		handler = h.Handler
		description = h.Description
		success = h.Success
		failure = h.Failure
		param = h.Param
	}
	// 保存至全局记录
	handlerMap[hid] = handler
	// 生成VirtHandler
	virtHandler := virtrouter.NewVirtHandler(hid, prefix, methods, description, success, failure, param)
	// 生成虚拟路由操作
	return virtrouter.NewVirtRouterHandler(name, virtHandler).ResetUse(middleware)
}

func handleWareUri(hw interface{}, methods []string, prefix string) string {
	add := "[" + prefix + "]"
	for _, m := range methods {
		add += "[" + m + "]"
	}
	t := reflect.ValueOf(hw).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(hw).Pointer()).Name() + add
	}
	return t.String() + add
}

// 路由执行前后的中间件
var (
	beforeMiddlewares = []string{}
	afterMiddlewares  = []string{}
)

func SetVirtBefore(middlewares []string) {
	beforeMiddlewares = middlewares
}

func SetVirtAfter(middlewares []string) {
	afterMiddlewares = middlewares
}

func getHandlerMap(id string) HandlerFunc {
	return handlerMap[id]
}
