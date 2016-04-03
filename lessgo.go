package lessgo

import (
	"net"
	"time"

	_ "github.com/lessgo/lessgo/_fixture"
	"github.com/lessgo/lessgo/engine"
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
		registerConfig()
		registerMime()
		l := &Lessgo{
			Echo: New(),
		}
		// 初始化日志
		l.Echo.Logger().SetMsgChan(AppConfig.Log.AsyncChan)
		l.Echo.SetLogLevel(AppConfig.Log.Level)
		// 设置运行模式
		l.Echo.SetDebug(AppConfig.Debug)
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
	DefLessgo.Logger().Sys("> %s listening and serving %s on %v", AppConfig.AppName, h, c.Address)
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

func wrapMiddlewares(middleware []interface{}) []MiddlewareFunc {
	ms := make([]MiddlewareFunc, len(middleware))
	for i, m := range middleware {
		ms[i] = WrapMiddleware(m)
	}
	return ms
}
