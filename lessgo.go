/*
Package lessgo implements a simple, stable, efficient and flexible web framework for Go.

Author1: https://github.com/henrylee2cn
Author2: https://github.com/changyu72
*/
package lessgo

import (
	"errors"
	"os"
	"os/exec"
	"path"
	"runtime"
	"sync"
	"time"

	_ "github.com/lessgo/lessgo/_fixture"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/utils"
	"github.com/lessgo/lessgoext/uuid"
)

type Lessgo struct {
	*App
	*config

	//全局操作列表
	apiHandlers []*ApiHandler

	//全局中间件列表
	apiMiddlewares []*ApiMiddleware

	// 路由执行前后的中间件登记
	virtBefore  []*MiddlewareConfig //处理链中路由操作之前的中间件子链
	virtAfter   []*MiddlewareConfig //处理链中路由操作之后的中间件子链
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
	VERSION = "0.7.0"
	ADDRESS = "https://github.com/lessgo/lessgo"
)

var (
	// 初始化全局Lessgo实例
	lessgo = newLessgo()

	// 初始化全局App实例
	app = newApp()

	// 全局配置实例
	Config = newConfig()

	// 全局运行日志实例(来自数据库的日志除外)
	Log = func() logs.Logger {
		l := logs.NewLogger(1000)
		l.AddAdapter("console", "")
		l.AddAdapter("file", `{"filename":"`+LOG_FILE+`"}`)
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

// Session管理平台实例
func Sessions() *session.Manager {
	return app.Sessions()
}

// 设置请求的url不存在时的默认操作(内部有默认实现)
// 404 Not Found
func SetNotFound(fn func(*Context) error) {
	app.SetNotFound(fn)
}

// 设置请求的url存在但方法不被允许时的默认操作(内部有默认实现)
// 405 Method Not Allowed
func SetMethodNotAllowed(fn func(*Context) error) {
	app.SetMethodNotAllowed(fn)
}

// 设置请求的操作发生错误后的默认处理(内部有默认实现)
// 500 Internal Server Error
func SetInternalServerError(fn func(c *Context, err error, rcv interface{})) {
	app.SetInternalServerError(fn)
}

// 设置捆绑数据处理接口(内部有默认实现)
func SetBinder(b Binder) {
	app.SetBinder(b)
}

// 设置html模板处理接口(内部有默认实现)
func SetRenderer(r Renderer) {
	app.SetRenderer(r)
}

// 判断当前是否为调试模式
func Debug() bool {
	return app.Debug()
}

// 设置运行模式
func SetDebug(on bool) {
	app.SetDebug(on)
}

// 判断文件缓存是否开启
func CanMemoryCache() bool {
	return app.CanMemoryCache()
}

// 启用文件缓存
func EnableMemoryCache() {
	app.memoryCache.SetEnable(true)
}

// 关闭文件缓存
func DisableMemoryCache() {
	app.memoryCache.SetEnable(false)
}

// 主动刷新缓存文件
func RefreshMemoryCache() {
	app.memoryCache.TriggerScan()
}

// 获取已注册的操作列表
func Handlers() []*ApiHandler {
	return lessgo.apiHandlers
}

// 获取已注册的中间件列表
func Middlewares() []*ApiMiddleware {
	return lessgo.apiMiddlewares
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
	return app.RealRoutes()
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
func PreUse(middlewares ...interface{}) error {
	ms, err := WrapMiddlewareConfigs(middlewares)
	if err != nil {
		return err
	}
	lessgo.virtBefore = append(ms, lessgo.virtBefore...)
	return nil
}

// 插入到处理链中路由操作前一位的中间件(子链)
func BeforeUse(middlewares ...interface{}) error {
	ms, err := WrapMiddlewareConfigs(middlewares)
	if err != nil {
		return err
	}
	lessgo.virtBefore = append(lessgo.virtBefore, ms...)
	return nil
}

// 插入到处理链中路由操作后一位的中间件(子链)
func AfterUse(middlewares ...interface{}) error {
	ms, err := WrapMiddlewareConfigs(middlewares)
	if err != nil {
		return err
	}
	lessgo.virtAfter = append(ms, lessgo.virtAfter...)
	return nil
}

// 追加到处理链最末端的中间件(子链)
func SufUse(middlewares ...interface{}) error {
	ms, err := WrapMiddlewareConfigs(middlewares)
	if err != nil {
		return err
	}
	lessgo.virtAfter = append(lessgo.virtAfter, ms...)
	return nil
}

// 单独注册静态文件虚拟路由VirtFile(无法在Root()下使用)
func File(path, file string, middlewares ...interface{}) error {
	ms, err := WrapMiddlewareConfigs(middlewares)
	if err != nil {
		return err
	}
	for _, v := range lessgo.virtFiles {
		if v.Path == path {
			v.File = file
			v.Middlewares = ms
			return nil
		}
	}
	lessgo.virtFiles = append(lessgo.virtFiles, &VirtFile{
		Path:        path,
		File:        file,
		Middlewares: ms,
	})
	return nil
}

// 单独注册静态目录虚拟路由VirtStatic(无法在Root()下使用)
func Static(prefix, root string, middlewares ...interface{}) error {
	ms, err := WrapMiddlewareConfigs(middlewares)
	if err != nil {
		return err
	}
	for _, v := range lessgo.virtStatics {
		if v.Prefix == prefix {
			v.Root = root
			v.Middlewares = ms
			return nil
		}
	}
	lessgo.virtStatics = append(lessgo.virtStatics, &VirtStatic{
		Prefix:      prefix,
		Root:        root,
		Middlewares: ms,
	})
	return nil
}

// 清空用户添加到处理链中路由操作前的所有中间件(子链)
func ResetBefore() {
	lessgo.virtBefore = lessgo.virtBefore[:0]
	registerBefore()
}

// 清空用户添加到处理链中路由操作后的所有中间件(子链)
func ResetAfter() {
	lessgo.virtAfter = lessgo.virtAfter[:0]
	registerAfter()
}

// 清空用户单独注册静态目录虚拟路由VirtFile
func ResetStatics() {
	lessgo.virtStatics = lessgo.virtStatics[:0]
	registerStatics()
}

// 清空用户单独注册静态文件虚拟路由VirtFile
func ResetFiles() {
	lessgo.virtFiles = lessgo.virtFiles[:0]
	registerFiles()
}

// 创建静态目录服务的操作(用于在Root()下)
func StaticFunc(root string) HandlerFunc {
	return func(c *Context) error {
		return c.File(path.Join(root, c.PathParamByIndex(0)))
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
	case func(*Context) error:
		x = HandlerFunc(t)
	default:
		panic("[" + utils.ObjectName(h) + "] can not be converted to MiddlewareFunc.")
	}
	return func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if err := x(c); err != nil {
				return err
			}
			return next(c)
		}
	}
}

// 自动转换某些允许的对象为中间件配置类型.
func WrapMiddlewareConfigs(middlewares []interface{}) ([]*MiddlewareConfig, error) {
	ms := make([]*MiddlewareConfig, len(middlewares))
	for i, o := range middlewares {
		switch m := o.(type) {
		case *MiddlewareConfig:
			ms[i] = m
		case MiddlewareConfig:
			ms[i] = &m
		case *ApiMiddleware:
			ms[i] = m.init().NewMiddlewareConfig()
		case ApiMiddleware:
			ms[i] = m.init().NewMiddlewareConfig()
		default:
			return ms, errors.New("[" + utils.ObjectName(m) + "] can not be converted to *MiddlewareConfig.")
		}
	}
	return ms, nil
}

// 重建底层真实路由
func ReregisterRouter() {
	var err error
	// 检查路由操作执行前后，中间件配置的可用性
	if err = isExistMiddlewares(lessgo.virtBefore...); err != nil {
		Log.Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err = isExistMiddlewares(lessgo.virtAfter...); err != nil {
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
	app.lock.Lock()
	defer app.lock.Unlock()

	// 从虚拟路由创建真实路由
	app.cleanRouter()
	app.beforeUse(getMiddlewareFuncs(lessgo.virtBefore)...)
	app.afterUse(getMiddlewareFuncs(lessgo.virtAfter)...)
	group := app.group(
		lessgo.virtRouter.Prefix,
		getMiddlewareFuncs(lessgo.virtRouter.Middlewares)...,
	)
	for _, child := range lessgo.virtRouter.Children {
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
	// 尝试设置系统默认通用操作
	tryRegisterDefaultHandler()

	// 添加系统预设的路由操作前的中间件
	registerBefore()

	// 添加系统预设的路由操作后的中间件
	registerAfter()

	// 添加系统预设的静态目录虚拟路由
	registerStatics()

	// 添加系统预设的静态文件虚拟路由
	registerFiles()

	// 从数据库初始化虚拟路由
	initVirtRouterConfig()

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
	app.run(
		Config.Listen.Address,
		tlsCertfile,
		tlsKeyfile,
		time.Duration(Config.Listen.ReadTimeout),
		time.Duration(Config.Listen.WriteTimeout),
		Config.Listen.Graceful,
	)
}
