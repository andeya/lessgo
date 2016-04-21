/*
Package lessgo implements a simple, stable, efficient and flexible web framework for Go.

Author1: https://github.com/henrylee2cn
Author2: https://github.com/changyu72
*/
package lessgo

import (
	"fmt"
	"net"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-xorm/xorm"

	_ "github.com/lessgo/lessgo/_fixture"
	"github.com/lessgo/lessgo/dbservice"
	"github.com/lessgo/lessgo/engine"
	"github.com/lessgo/lessgo/logs"
)

type (
	lessgo struct {
		app       *Echo
		AppConfig *Config
		dbService *dbservice.DBService

		VirtRouter      *VirtRouter              //虚拟路由
		virtMiddlewares map[string]MiddlewareObj //登记虚拟中间件
		virtBefore      []string                 //登记路由执行前虚拟中间件
		virtAfter       []string                 //登记路由执行后虚拟中间件

		home         string //根路径"/"对应的url
		serverEnable bool   //服务是否启用
		lock         sync.RWMutex
	}
	NewServer func(engine.Config) engine.Server
)

const (
	NAME    = "Lessgo"
	VERSION = "0.4.0"
	ADDRESS = "https://github.com/lessgo/lessgo"
)

const (
	MB = 1 << 20
)

// 初始化全局Lessgo实例
var DefLessgo = newLessgo()

/*
 * 获取lessgo全局实例
 */
func Lessgo() *lessgo {
	return DefLessgo
}

/*
 * 设置主页
 */
func SetHome(homeurl string) {
	if DefLessgo.AppConfig.RouterCaseSensitive {
		DefLessgo.home = homeurl
	} else {
		DefLessgo.home = strings.ToLower(homeurl)
	}
}

/*
 * 返回设置的主页
 */
func GetHome() string {
	return DefLessgo.home
}

/*
 * 开启网站服务
 */
func EnableServer() {
	DefLessgo.lock.Lock()
	DefLessgo.serverEnable = true
	DefLessgo.lock.Unlock()
}

/*
 * 关闭网站服务
 */
func DisableServer() {
	DefLessgo.lock.Lock()
	DefLessgo.serverEnable = false
	DefLessgo.lock.Unlock()
}

/*
 * 查询网站服务状态
 */
func ServerEnable() bool {
	DefLessgo.lock.RLock()
	defer DefLessgo.lock.RUnlock()
	return DefLessgo.serverEnable
}

/*
 * 运行服务
 */
func Run(server NewServer, listener ...net.Listener) {
	// 初始化全局session
	checkHooks(registerSession())
	// 从数据库同步虚拟路由
	syncVirtRouter()
	// 重建路由
	ReregisterRouter()

	// 配置服务器引擎
	c := engine.Config{
		Address:      AppConfig.Listen.Address,
		ReadTimeout:  time.Duration(AppConfig.Listen.ReadTimeout),
		WriteTimeout: time.Duration(AppConfig.Listen.WriteTimeout),
	}
	h := "HTTP"
	if AppConfig.Listen.EnableHTTPS {
		h = "HTTPS"
		c.TLSKeyfile = AppConfig.Listen.HTTPSKeyFile
		c.TLSCertfile = AppConfig.Listen.HTTPSCertFile
	}
	if len(listener) > 0 && listener[0] != nil {
		c.Listener = listener[0]
	}

	// 开启最大核心数运行
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 启动服务
	mode := "release"
	if AppConfig.Debug {
		mode = "debug"
	}
	Logger().Sys("> %s listening and serving %s on %v (%s-mode)", AppConfig.AppName, h, c.Address, mode)
	DefLessgo.app.Run(server(c))
}

/*
 * 数据库引擎
 */

// 获取默认数据库引擎
func DefaultDB() *xorm.Engine {
	return DefLessgo.dbService.DefaultDB()
}

// 获取指定数据库引擎
func GetDB(name string) (*xorm.Engine, bool) {
	return DefLessgo.dbService.GetDB(name)
}

/*
 * 打印实例
 */
func Logger() logs.Logger {
	return DefLessgo.app.Logger()
}

/*
 * 虚拟路由
 */

// 虚拟路由列表
func RouterList() []*VirtRouter {
	return DefLessgo.VirtRouter.Progeny()
}

// 在路由执行位置之前紧邻插入中间件队列
func BeforeUse(middleware ...string) {
	DefLessgo.virtBefore = append(DefLessgo.virtBefore, middleware...)
}

// 在路由执行位置之后紧邻插入中间件队列
func AfterUser(middleware ...string) {
	DefLessgo.virtAfter = append(middleware, DefLessgo.virtAfter...)
}

// 从根路由开始配置路由(必须在init()中调用)
func RootRouter(node ...*VirtRouter) *VirtRouter {
	DefLessgo.VirtRouter.AddChildren(node)
	return DefLessgo.VirtRouter
}

// 配置路由分组(必须在init()中调用)
func SubRouter(prefix, name string, node ...*VirtRouter) *VirtRouter {
	return NewVirtRouterGroup(prefix, name).AddChildren(node)
}

// 配置操作(必须在init()中调用)
func Get(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	return route([]string{GET}, prefix, name, descHandlerOrhandler, middleware)
}
func Head(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	return route([]string{HEAD}, prefix, name, descHandlerOrhandler, middleware)
}
func Options(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	return route([]string{OPTIONS}, prefix, name, descHandlerOrhandler, middleware)
}
func Patch(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	return route([]string{PATCH}, prefix, name, descHandlerOrhandler, middleware)
}
func Post(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	return route([]string{POST}, prefix, name, descHandlerOrhandler, middleware)
}
func Put(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	return route([]string{PUT}, prefix, name, descHandlerOrhandler, middleware)
}
func Trace(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	return route([]string{TRACE}, prefix, name, descHandlerOrhandler, middleware)
}
func Any(prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	return route([]string{CONNECT, DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT, TRACE}, prefix, name, descHandlerOrhandler, middleware)
}
func Match(methods []string, prefix, name string, descHandlerOrhandler interface{}, middleware ...string) *VirtRouter {
	if len(methods) == 0 {
		Logger().Error("The method can not be empty: %v", name)
	}
	return route(methods, prefix, name, descHandlerOrhandler, middleware)
}

/*
 * 重建真实路由
 */
func ReregisterRouter() {
	DefLessgo.app.lock.Lock()
	defer DefLessgo.app.lock.Unlock()
	registerVirtRouter()
	registerPreUse()
	registerSufUse()
	registerStaticRouter()
}

/*
 * 中间件
 */

// 获取已注册的中间件列表
func Middlewares() map[string]MiddlewareObj {
	return DefLessgo.virtMiddlewares
}

// 注册虚拟路由中使用的中间件，须在init()中调用
func RegMiddleware(name, description string, middleware interface{}) error {
	if _, ok := DefLessgo.virtMiddlewares[name]; ok {
		err := fmt.Errorf("RegisterMiddlewareFunc called twice for middleware %v.", name)
		Logger().Error("%v", err)
		return err
	}
	DefLessgo.virtMiddlewares[name] = MiddlewareObj{
		Name:           name,
		Description:    description,
		MiddlewareFunc: WrapMiddleware(middleware),
	}
	return nil
}
