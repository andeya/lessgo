package lessgo

import (
	"net"
	"path"
	"time"

	_ "github.com/lessgo/lessgo/_fixture"
	"github.com/lessgo/lessgo/engine"
	"github.com/lessgo/lessgo/logs"
)

type (
	Lessgo struct {
		*Echo
	}
	NewServer func(engine.Config) engine.Server
)

const (
	NAME    = "Lessgo"
	VERSION = "0.4.0"
	ADDRESS = "https://github.com/lessgo/lessgo"
)

var (
	DefLessgo = func() *Lessgo {
		printInfo()
		registerAppConfig()
		registerDBConfig()
		registerMime()
		l := &Lessgo{
			Echo: New(),
		}
		// 初始化日志
		l.Echo.Logger().SetMsgChan(AppConfig.Log.AsyncChan)
		l.Echo.SetLogLevel(AppConfig.Log.Level)
		// 设置运行模式
		l.Echo.SetDebug(AppConfig.Debug)
		// 设置静态资源缓存刷新频率
		l.Echo.SetMemoryCache(NewMemoryCache(time.Duration(AppConfig.FileCacheSecond) * time.Second))
		// 设置渲染接口
		l.Echo.SetRenderer(NewPongo2Render(AppConfig.Debug))
		// 设置大小写敏感
		l.Echo.SetCaseSensitive(AppConfig.RouterCaseSensitive)
		// 设置上传文件允许的最大尺寸
		engine.MaxMemory = AppConfig.MaxMemory
		return l
	}()
)

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
	// 启动服务
	mode := "release"
	if AppConfig.Debug {
		mode = "debug"
	}
	DefLessgo.Logger().Sys("> %s listening and serving %s on %v (%s-mode)", AppConfig.AppName, h, c.Address, mode)
	DefLessgo.Run(server(c))
}

// 在路由执行位置之前紧邻插入中间件队列
func Before(middleware ...interface{}) {
	DefLessgo.Echo.BeforeUse(wrapMiddlewares(middleware)...)
}

// 在路由执行位置之后紧邻插入中间件队列
func After(middleware ...interface{}) {
	DefLessgo.Echo.AfterUse(wrapMiddlewares(middleware)...)
}

// 重建真实路由
func ResetRealRoute() {
	if err := middlewareExistCheck(DefDynaRouter); err != nil {
		DefLessgo.Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}

	DefLessgo.Echo.lock.Lock()
	defer DefLessgo.Echo.lock.Unlock()

	defer func() {
		// 创建静态路由
		staticRoute()
	}()

	// 创建动态路由
	DefLessgo.Echo.router = NewRouter(DefLessgo.Echo)
	DefLessgo.Echo.middleware = []MiddlewareFunc{DefLessgo.Echo.router.Process}
	DefLessgo.Echo.head = DefLessgo.Echo.pristineHead
	registerRootMiddlewares()
	DefLessgo.Echo.BeforeUse(getMiddlewares(beforeMiddlewares)...)
	DefLessgo.Echo.AfterUse(getMiddlewares(afterMiddlewares)...)
	for _, child := range DefDynaRouter.Children {
		var group *Group
		for _, d := range child.Tree() {
			mws := getMiddlewares(d.Middlewares)
			switch d.Type {
			case GROUP:
				if group == nil {
					group = DefLessgo.Echo.Group(d.Prefix, mws...)
					break
				}
				group = group.Group(d.Prefix, mws...)
			case HANDLER:
				if group == nil {
					DefLessgo.Echo.Match(d.Methods, path.Join(d.Prefix, d.Param), handlerFuncMap[d.Handler], mws...)
					break
				}
				group.Match(d.Methods, path.Join(d.Prefix, d.Param), handlerFuncMap[d.Handler], mws...)
			}
		}
	}
}

func Logger() logs.Logger {
	return DefLessgo.Echo.Logger()
}
