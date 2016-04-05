/*
Package lessgo implements a simple, stable, efficient and flexible web framework for Go.

Author1: https://github.com/henrylee2cn
Author2: https://github.com/changyu72
*/
package lessgo

import (
	"net"
	"path"
	"strings"
	"sync"
	"time"

	_ "github.com/lessgo/lessgo/_fixture"
	"github.com/lessgo/lessgo/engine"
	"github.com/lessgo/lessgo/logs"
)

type (
	Lessgo struct {
		*Echo
		AppConfig    *Config
		home         string // 根路径"/"对应的url
		serverEnable bool   // 服务是否启用
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

var (
	DefLessgo = func() *Lessgo {
		printInfo()
		registerAppConfig()
		registerDBConfig()
		registerMime()
		l := &Lessgo{
			Echo:         New(),
			AppConfig:    AppConfig,
			home:         "/",
			serverEnable: true,
		}
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
		return l
	}()
)

// 设置主页
func SetHome(homeurl string) {
	if DefLessgo.AppConfig.RouterCaseSensitive {
		DefLessgo.home = homeurl
	} else {
		DefLessgo.home = strings.ToLower(homeurl)
	}
}

// 查询主页
func Home() string {
	return DefLessgo.home
}

// 开启网站服务
func EnableServer() {
	DefLessgo.lock.Lock()
	DefLessgo.serverEnable = true
	DefLessgo.lock.Unlock()
}

// 关闭网站服务
func DisableServer() {
	DefLessgo.lock.Lock()
	DefLessgo.serverEnable = false
	DefLessgo.lock.Unlock()
}

// 查询网站服务状态
func ServerEnable() bool {
	DefLessgo.lock.RLock()
	defer DefLessgo.lock.RUnlock()
	return DefLessgo.serverEnable
}

// 运行服务
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
		if !child.Enable {
			continue
		}
		var group *Group
		for _, d := range child.Tree() {
			if !d.Enable {
				continue
			}
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
					if d.Prefix == "/index" {
						DefLessgo.Echo.Match(d.Methods, path.Join("/", d.Param), handlerFuncMap[d.Handler], mws...)
					}
					DefLessgo.Echo.Match(d.Methods, path.Join(d.Prefix, d.Param), handlerFuncMap[d.Handler], mws...)
					break
				}
				if d.Prefix == "/index" {
					group.Match(d.Methods, path.Join("/", d.Param), handlerFuncMap[d.Handler], mws...)
				}
				group.Match(d.Methods, path.Join(d.Prefix, d.Param), handlerFuncMap[d.Handler], mws...)
			}
		}
	}
}

func Logger() logs.Logger {
	return DefLessgo.Echo.Logger()
}
