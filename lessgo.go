/*
Package lessgo implements a simple, stable, efficient and flexible web framework for Go.

Author1: https://github.com/henrylee2cn
Author2: https://github.com/changyu72
*/
package lessgo

import (
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
		*Echo
		AppConfig    *Config
		home         string // 根路径"/"对应的url
		serverEnable bool   // 服务是否启用
		DBAccess     *dbservice.DBAccess
		VirtRouter   *VirtRouter
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
var DefLessgo = func() *lessgo {
	printInfo()
	registerAppConfig()
	registerDBConfig()
	registerMime()
	l := &lessgo{
		Echo:         New(),
		AppConfig:    AppConfig,
		home:         "/",
		serverEnable: true,
	}
	// 初始化全局虚拟路由
	l.VirtRouter, _ = NewVirtRouterRoot()
	// 初始化日志
	l.Echo.Logger().SetMsgChan(AppConfig.Log.AsyncChan)
	l.Echo.SetLogLevel(AppConfig.Log.Level)
	// 设置运行模式
	l.Echo.SetDebug(AppConfig.Debug)
	// 设置静态资源缓存
	l.Echo.SetMemoryCache(NewMemoryCache(
		AppConfig.FileCache.SingleFileAllowMB*MB,
		AppConfig.FileCache.MaxCapMB*MB,
		time.Duration(AppConfig.FileCache.CacheSecond)*time.Second),
	)
	// 设置渲染接口
	l.Echo.SetRenderer(NewPongo2Render(AppConfig.Debug))
	// 设置大小写敏感
	l.Echo.SetCaseSensitive(AppConfig.RouterCaseSensitive)
	// 设置上传文件允许的最大尺寸
	engine.MaxMemory = AppConfig.MaxMemoryMB * MB
	// 配置数据库
	l.DBAccess = newDBAccess()
	return l
}()

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
	checkHooks(registerRouter())
	checkHooks(registerSession())

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
	DefLessgo.Logger().Sys("> %s listening and serving %s on %v (%s-mode)", AppConfig.AppName, h, c.Address, mode)
	DefLessgo.Run(server(c))
}

/*
 * 获取默认数据库引擎
 */
func DefaultDB() *xorm.Engine {
	return DefLessgo.DBAccess.DefaultDB()
}

/*
 * 获取全部数据库引擎列表
 */
func DBList() map[string]*xorm.Engine {
	return DefLessgo.DBAccess.DBList()
}

/*
 * 设置默认数据库引擎
 */
func SetDefaultDB(name string) error {
	return DefLessgo.DBAccess.SetDefaultDB(name)
}

/*
 * 获取指定数据库引擎
 */
func GetDB(name string) (*xorm.Engine, bool) {
	return DefLessgo.DBAccess.GetDB(name)
}

/*
 * 返回打印实例
 */
func Logger() logs.Logger {
	return DefLessgo.Echo.Logger()
}

/*
 * 虚拟路由列表
 */
func VirtRouterList() []*VirtRouter {
	return DefLessgo.VirtRouter.Progeny()
}

/*
 * 注册虚拟路由
 */

// 路由执行前后的中间件
var (
	beforeMiddlewares = []string{
		"检查网站是否开启",
		"自动匹配home页面",
		"运行时请求日志",
		"异常恢复",
		"跨域",
	}
	afterMiddlewares = []string{}
)

// 在路由执行位置之前紧邻插入中间件队列
func BeforeUse(middleware ...string) {
	beforeMiddlewares = append(beforeMiddlewares, middleware...)
}

// 在路由执行位置之后紧邻插入中间件队列
func AfterUser(middleware ...string) {
	afterMiddlewares = append(middleware, afterMiddlewares...)
}

// 必须在init()中调用
// 从根路由开始配置路由
func RootRouter(node ...*VirtRouter) *VirtRouter {
	DefLessgo.VirtRouter.AddChildren(node)
	return DefLessgo.VirtRouter
}

// 必须在init()中调用
// 配置路由分组
func SubRouter(prefix, name string, node ...*VirtRouter) *VirtRouter {
	return NewVirtRouterGroup(prefix, name).AddChildren(node)
}

// 必须在init()中调用
// 配置操作
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
		DefLessgo.logger.Error("The method can not be empty: %v", name)
	}
	return route(methods, prefix, name, descHandlerOrhandler, middleware)
}

/*
 * 强制注册基础路由
 */

func BaseRouter(path string, methods []string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	baseRouterLock.Lock()
	baseRouterMap[path] = &baseRouter{
		Path:       path,
		Methods:    methods,
		Handler:    handler,
		Middleware: middleware,
	}
	baseRouterLock.Unlock()
}

func StaticBaseRouter(prefix, root string, middleware ...MiddlewareFunc) {
	baseRouterLock.Lock()
	staticBaseRouterMap[prefix] = &staticBaseRouter{
		Prefix:     prefix,
		Root:       root,
		Middleware: middleware,
	}
	baseRouterLock.Unlock()
}

func FileBaseRouter(path, file string, middleware ...MiddlewareFunc) {
	baseRouterLock.Lock()
	fileBaseRouterMap[path] = &fileBaseRouter{
		Path:       path,
		File:       file,
		Middleware: middleware,
	}
	baseRouterLock.Unlock()
}

/*
 * 重建真实路由
 */
func ResetRealRoute() {
	if err := middlewareCheck(beforeMiddlewares); err != nil {
		DefLessgo.Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err := middlewareCheck(afterMiddlewares); err != nil {
		DefLessgo.Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err := middlewareCheck(DefLessgo.VirtRouter.AllMiddleware()); err != nil {
		DefLessgo.Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}

	DefLessgo.Echo.lock.Lock()
	defer DefLessgo.Echo.lock.Unlock()

	defer func() {
		// 强制注册基础路由
		registerBaseRouter()
	}()

	// 创建动态路由
	DefLessgo.Echo.router = NewRouter(DefLessgo.Echo)
	DefLessgo.Echo.middleware = []MiddlewareFunc{DefLessgo.Echo.router.Process}
	DefLessgo.Echo.head = DefLessgo.Echo.pristineHead
	DefLessgo.Echo.BeforeUse(getMiddlewares(beforeMiddlewares)...)
	DefLessgo.Echo.AfterUse(getMiddlewares(afterMiddlewares)...)
	group := DefLessgo.Echo.Group(
		DefLessgo.VirtRouter.Prefix(),
		getMiddlewares(DefLessgo.VirtRouter.Middleware())...,
	)
	for _, child := range DefLessgo.VirtRouter.Children() {
		child.route(group)
	}
}
