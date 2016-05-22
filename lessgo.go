/*
Package lessgo implements a simple, stable, efficient and flexible web framework for Go.

Author1: https://github.com/henrylee2cn
Author2: https://github.com/changyu72
*/
package lessgo

import (
	"os"
	"os/exec"
	"path"
	"runtime"
	"sync"
	"time"

	"github.com/go-xorm/xorm"

	_ "github.com/lessgo/lessgo/_fixture"
	"github.com/lessgo/lessgo/dbservice"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/utils"
	"github.com/lessgo/lessgo/utils/uuid"
)

type Lessgo struct {
	*App
	*config
	dbService *dbservice.DBService

	//全局操作列表
	apiHandlers []*ApiHandler

	//全局中间件列表
	apiMiddlewares []*ApiMiddleware

	// 路由执行前后的中间件登记
	before      []*MiddlewareConfig //处理链中路由操作之前的中间件子链
	after       []*MiddlewareConfig //处理链中路由操作之后的中间件子链
	virtStatics []*VirtStatic       //单独注册的静态目录虚拟路由(无法在Root()下使用)
	virtFiles   []*VirtFile         //单独注册的静态文件虚拟路由(无法在Root()下使用)
	// 用于构建最终真实路由的虚拟路由；
	// 初始值为源码中定义的路由，之后追加配置中定义的路由；
	// 配置路由为空时，复制源码中定义的路由到配置路由；
	// 再次运行时，直接读取配置路由覆盖自身；
	// 源码因修改产生冲突时，以源码路由为主；
	// 修改配置路由时，不允许和源码路由冲突；
	// 源码路由只允许在配置中增加子节点。
	virtRouter *VirtRouter

	home         string //根路径"/"对应的url
	serverEnable bool   //服务是否启用
	lock         sync.RWMutex
}

const (
	NAME    = "Lessgo"
	VERSION = "0.6.0"
	ADDRESS = "https://github.com/lessgo/lessgo"
)

const (
	MB = 1 << 20
)

var (
	// 初始化全局Lessgo实例
	lessgo = newLessgo()

	// 全局配置实例
	Config = newConfig()

	// 全局运行日志实例(来自数据库的日志除外)
	Log = func() logs.Logger {
		l := logs.NewLogger(1000)
		l.AddAdapter("console", "")
		l.AddAdapter("file", `{"filename":"logger/lessgo.log"}`)
		return l
	}()

	// 软件自身md5值
	Md5 = func() string {
		file, _ := exec.LookPath(os.Args[0])
		info, _ := os.Stat(file)
		return utils.MakeUnique(info.ModTime())
	}()
)

// 返回设置的主页
func GetHome() string {
	return lessgo.home
}

// 设置主页(内部已默认为"/")
func SetHome(homeurl string) {
	lessgo.home = homeurl
}

// 查询网站服务状态
func ServerEnable() bool {
	lessgo.lock.RLock()
	defer lessgo.lock.RUnlock()
	return lessgo.serverEnable
}

//  开启网站服务
func EnableServer() {
	lessgo.lock.Lock()
	lessgo.serverEnable = true
	lessgo.lock.Unlock()
}

// 关闭网站服务
func DisableServer() {
	lessgo.lock.Lock()
	lessgo.serverEnable = false
	lessgo.lock.Unlock()
}

// 获取默认数据库引擎
func DefaultDB() *xorm.Engine {
	return lessgo.dbService.DefaultDB()
}

// 获取指定数据库引擎
func GetDB(name string) (*xorm.Engine, bool) {
	return lessgo.dbService.GetDB(name)
}

// Session管理平台实例
func Sessions() *session.Manager {
	return lessgo.App.Sessions()
}

// 设置请求的url不存在时的默认操作(内部有默认实现)
// 404 Not Found
func SetNotFound(fn func(Context) error) {
	lessgo.App.SetNotFound(fn)
}

// 设置请求的url存在但方法不被允许时的默认操作(内部有默认实现)
// 405 Method Not Allowed
func SetMethodNotAllowed(fn func(Context) error) {
	lessgo.App.SetMethodNotAllowed(fn)
}

// 设置请求的操作发生错误后的默认处理(内部有默认实现)
// 500 Internal Server Error
func SetInternalServerError(fn func(error, Context)) {
	lessgo.App.SetInternalServerError(fn)
}

// 设置捆绑数据处理接口(内部有默认实现)
func SetBinder(b Binder) {
	lessgo.App.SetBinder(b)
}

// 设置html模板处理接口(内部有默认实现)
func SetRenderer(r Renderer) {
	lessgo.App.SetRenderer(r)
}

// 判断当前是否为调试模式
func Debug() bool {
	return lessgo.App.Debug()
}

// 设置运行模式
func SetDebug(on bool) {
	lessgo.App.SetDebug(on)
}

// 设置文件内存缓存功能(内部有默认实现)
func SetMemoryCache(m *MemoryCache) {
	lessgo.App.SetMemoryCache(m)
}

// 判断是否开启了文件内存缓存功能
func MemoryCacheEnable() bool {
	return lessgo.App.MemoryCacheEnable()
}

// 获取已注册的操作列表
func Handlers() []*ApiHandler {
	return lessgo.apiHandlers
}

// 获取已注册的中间件列表
func Middlewares() []*ApiMiddleware {
	return lessgo.apiMiddlewares
}

// 自动转换某些允许的函数为中间件函数.
func WrapMiddleware(h interface{}) MiddlewareFunc {
	var x HandlerFunc
	switch t := h.(type) {
	case MiddlewareFunc:
		return t
	case func(HandlerFunc) HandlerFunc:
		return MiddlewareFunc(t)
	case HandlerFunc:
		x = t
	case func(Context) error:
		x = HandlerFunc(t)
	default:
		panic("[" + utils.ObjectName(h) + "] can not be converted to MiddlewareFunc.")
	}
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			if err := x(c); err != nil {
				return err
			}
			return next(c)
		}
	}
}

// 返回当前虚拟的路由列表(不含单独注册的静态路由VirtFiles/VirtStatics)
func VirtRoutes() []*VirtRouter {
	return lessgo.virtRouter.Progeny()
}

// 返回单独注册的静态文件虚拟路由列表(非Root()下注册)
func VirtFiles() []*VirtFile {
	return lessgo.virtFiles
}

// 返回单独注册的静态目录虚拟路由列表(非Root()下注册)
func VirtStatics() []*VirtStatic {
	return lessgo.virtStatics
}

// 返回底层注册的路由列表(全部真实注册的路由)
func RealRoutes() []Route {
	return lessgo.App.RealRoutes()
}

// 虚拟路由根节点
func RootRouter() *VirtRouter {
	return lessgo.virtRouter
}

// 操作列表(禁止修改)
func ApiHandlerList() []*ApiHandler {
	return lessgo.apiHandlers
}

// 添加到处理链最前端的中间件(子链)
func PreUse(middleware ...*MiddlewareConfig) {
	lessgo.before = append(middleware, lessgo.before...)
}

// 插入到处理链中路由操作前一位的中间件(子链)
func BeforeUse(middleware ...*MiddlewareConfig) {
	lessgo.before = append(lessgo.before, middleware...)
}

// 插入到处理链中路由操作后一位的中间件(子链)
func AfterUse(middleware ...*MiddlewareConfig) {
	lessgo.after = append(middleware, lessgo.after...)
}

// 追加到处理链最末端的中间件(子链)
func SufUse(middleware ...*MiddlewareConfig) {
	lessgo.after = append(lessgo.after, middleware...)
}

// 单独注册静态文件虚拟路由VirtFile(无法在Root()下使用)
func File(path, file string, middleware ...*MiddlewareConfig) {
	lessgo.virtFiles = append(lessgo.virtFiles, &VirtFile{
		Path:        path,
		File:        file,
		Middlewares: middleware,
	})
}

// 单独注册静态目录虚拟路由VirtStatic(无法在Root()下使用)
func Static(prefix, root string, middleware ...*MiddlewareConfig) {
	lessgo.virtStatics = append(lessgo.virtStatics, &VirtStatic{
		Prefix:      prefix,
		Root:        root,
		Middlewares: middleware,
	})
}

// 创建静态目录服务的操作(用于在Root()下)
func StaticFunc(root string) HandlerFunc {
	return func(c Context) error {
		return c.File(path.Join(root, c.P(0)))
	}
}

// 从根路由开始配置虚拟路由(必须在init()中调用)
func Root(nodes ...*VirtRouter) {
	var err error
	for _, node := range nodes {
		if node == nil {
			continue
		}
		err = lessgo.virtRouter.addChild(node)
		if err != nil {
			Log.Error("%v", err)
		}
	}
}

// 配置虚拟路由分组(必须在init()中调用)
func Branch(prefix, desc string, nodes ...*VirtRouter) *VirtRouter {
	parent := NewGroupVirtRouter(prefix, desc)
	parent.Dynamic = false
	var err error
	for _, node := range nodes {
		if node == nil {
			continue
		}
		err = parent.addChild(node)
		if err != nil {
			Log.Error("%v", err)
		}
	}
	return parent
}

// 配置虚拟路由操作(必须在init()中调用)
func Leaf(prefix string, apiHandler *ApiHandler, middlewares ...*ApiMiddleware) *VirtRouter {
	prefix = cleanPrefix(prefix)
	ms := make([]*MiddlewareConfig, len(middlewares))
	for i, m := range middlewares {
		m.init()
		ms[i] = m.NewMiddlewareConfig()
	}
	vr := &VirtRouter{
		Id:          uuid.New().String(),
		Type:        HANDLER,
		Prefix:      prefix,
		Enable:      true,
		Dynamic:     false,
		Middlewares: ms,
		apiHandler:  apiHandler.init(),
		Hid:         apiHandler.id,
	}
	return vr
}

// 重建底层真实路由
func ReregisterRouter() {
	var err error
	// 检查路由操作执行前后，中间件配置的可用性
	if err = isExistMiddlewares(lessgo.before...); err != nil {
		Log.Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err = isExistMiddlewares(lessgo.after...); err != nil {
		Log.Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	for _, v := range lessgo.virtFiles {
		if err = isExistMiddlewares(v.Middlewares...); err != nil {
			Log.Error("Create/Recreate the router is faulty: %v", err)
			return
		}
	}
	for _, v := range lessgo.virtStatics {
		if err = isExistMiddlewares(v.Middlewares...); err != nil {
			Log.Error("Create/Recreate the router is faulty: %v", err)
			return
		}
	}

	// 阻塞所有产生的请求
	lessgo.App.lock.Lock()
	defer lessgo.App.lock.Unlock()

	// 从虚拟路由创建真实路由
	lessgo.App.cleanRouter()
	lessgo.App.beforeUse(getMiddlewareFuncs(lessgo.before)...)
	lessgo.App.afterUse(getMiddlewareFuncs(lessgo.after)...)
	group := lessgo.App.group(
		lessgo.virtRouter.Prefix,
		getMiddlewareFuncs(lessgo.virtRouter.Middlewares)...,
	)
	for _, child := range lessgo.virtRouter.Children() {
		child.route(group)
	}

	// 从单独的静态文件虚拟路由注册真实路由
	for _, v := range lessgo.virtFiles {
		v.route()
	}
	// 从单独的静态目录虚拟路由注册真实路由
	for _, v := range lessgo.virtStatics {
		v.route()
	}
}

// 运行服务
func Run() {
	// 添加系统预设的中间件
	registerMiddleware()
	// 添加系统预设的静态虚拟路由
	registerStaticRouter()
	// 从数据库初始化虚拟路由
	initVirtRouterFromDB()
	// 重建路由
	ReregisterRouter()

	// 开启最大核心数运行
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 配置服务器引擎
	var (
		tlsCertfile string
		tlsKeyfile  string
		mode        string
		graceful    string
		protocol    = "HTTP"
	)
	if Config.Listen.EnableHTTPS {
		protocol = "HTTPS"
		tlsCertfile = Config.Listen.HTTPSCertFile
		tlsKeyfile = Config.Listen.HTTPSKeyFile
	}
	if Config.Debug {
		mode = "debug"
	} else {
		mode = "release"
	}
	if Config.Listen.Graceful {
		graceful = "(enable-graceful-restart)"
	} else {
		graceful = "(disable-graceful-restart)"
	}

	Log.Sys("> %s listening and serving %s on %v (%s-mode) %v", Config.AppName, protocol, Config.Listen.Address, mode, graceful)

	// 启动服务
	lessgo.App.run(
		Config.Listen.Address,
		tlsCertfile,
		tlsKeyfile,
		time.Duration(Config.Listen.ReadTimeout),
		time.Duration(Config.Listen.WriteTimeout),
		Config.Listen.Graceful,
	)
}
