package lessgo

import (
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/lessgo/lessgo/utils"
	"github.com/lessgo/lessgo/virtrouter"
)

var (
	// 全局虚拟路由
	DefVirtRouter, _ = virtrouter.NewRootVirtRouter()
	// 全部handler及其id
	handlerMap = map[string]HandlerFunc{}
)

// 返回虚拟路由列表
func VirtRouterProgeny() []*virtrouter.VirtRouter {
	return DefVirtRouter.Progeny()
}

// 快速返回指定url对于的虚拟路由节点
func GetVirtRouter(u string) (*virtrouter.VirtRouter, bool) {
	return virtrouter.GetVirtRouter(u)
}

// 必须在init()中调用
// 从根路由开始配置路由
func RootRouter(node ...*virtrouter.VirtRouter) {
	DefVirtRouter.AddChildren(node)
}

// 必须在init()中调用
// 配置路由分组
func SubRouter(prefix, name string, node ...*virtrouter.VirtRouter) *virtrouter.VirtRouter {
	return virtrouter.NewVirtRouter(virtrouter.GROUP, prefix, name, nil).AddChildren(node)
}

// 必须在init()中调用
// 配置操作
func Get(name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	return route([]string{GET}, name, descHandlerOrhandler, suffix...)
}
func Head(name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	return route([]string{HEAD}, name, descHandlerOrhandler, suffix...)
}
func Options(name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	return route([]string{OPTIONS}, name, descHandlerOrhandler, suffix...)
}
func Patch(name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	return route([]string{PATCH}, name, descHandlerOrhandler, suffix...)
}
func Post(name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	return route([]string{POST}, name, descHandlerOrhandler, suffix...)
}
func Put(name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	return route([]string{PUT}, name, descHandlerOrhandler, suffix...)
}
func Trace(name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	return route([]string{TRACE}, name, descHandlerOrhandler, suffix...)
}
func Any(name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	return route([]string{CONNECT, DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT, TRACE}, name, descHandlerOrhandler, suffix...)
}
func Match(methods []string, name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	if len(methods) == 0 {
		DefLessgo.logger.Error("The method can not be empty: %v", name)
	}
	return route(methods, name, descHandlerOrhandler, suffix...)
}

func route(methods []string, name string, descHandlerOrhandler interface{}, suffix ...string) *virtrouter.VirtRouter {
	sort.Strings(methods)
	hid := handleWareUri(descHandlerOrhandler)
	var _suffix string
	if len(suffix) > 0 {
		_suffix = suffix[0]
	}

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
	virtHandler, _ := virtrouter.NewVirtHandler(hid, _suffix, methods, description, success, failure, param)

	ns := strings.Split(hid, ".")
	n := strings.TrimSuffix(ns[len(ns)-1], "Handle")
	prefix := "/" + utils.SnakeString(n)

	return virtrouter.NewVirtRouter(virtrouter.HANDLER, prefix, name, virtHandler)
}

func handleWareUri(hw interface{}) string {
	t := reflect.ValueOf(hw).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(hw).Pointer()).Name()
	}
	return t.String()
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
