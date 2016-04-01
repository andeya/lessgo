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

var (
	DefLessgo = func() *Lessgo {
		registerConfig()
		registerMime()
		l := &Lessgo{
			Echo: New(),
		}
		// 设置运行模式
		l.Echo.SetDebug(AppConfig.Debug)
		// 设置上传文件允许的最大尺寸
		engine.MaxMemory = AppConfig.MaxMemory
		// 初始化日志
		l.Echo.SetLogLevel(AppConfig.Log.Level)
		l.Echo.LogFuncCallDepth(AppConfig.Log.FileLineNum)
		l.Echo.LogAsync(AppConfig.Log.AsyncChan)
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
	if AppConfig.Listen.EnableHTTPS {
		c.TLSKeyfile = AppConfig.Listen.HTTPSKeyFile
		c.TLSCertfile = AppConfig.Listen.HTTPSCertFile
	}
	if len(listener) > 0 && listener[0] != nil {
		c.Listener = listener[0]
	}
	// 启动服务
	DefLessgo.Run(server(c))
}

func rootGroup() {
	// DefLessgo.Echo.Pre(Logger())
	DefLessgo.Echo.Suf()
}

func checkHooks(err error) {
	if err == nil {
		return
	}
	DefLessgo.Echo.Logger().Fatal("%v", err)
}
