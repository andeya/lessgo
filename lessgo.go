package lessgo

import (
	"net"
	"time"

	"github.com/lessgo/lessgo/engine"
	// "github.com/lessgo/lessgo/session"
)

type (
	Lessgo struct {
		*Echo
		Config *Config
	}
	NewServer func(engine.Config) engine.Server
)

var (
	DefLessgo = &Lessgo{Echo: New()}
)

func Run(ns NewServer, listener ...net.Listener) {
	checkHooks(registerMime())
	checkHooks(registerRouter())
	engine.MaxMemory = DefLessgo.Config.MaxMemory
	c := engine.Config{
		Address:      DefLessgo.Config.Listen.Address,
		ReadTimeout:  time.Duration(DefLessgo.Config.Listen.ReadTimeout),
		WriteTimeout: time.Duration(DefLessgo.Config.Listen.WriteTimeout),
	}
	if DefLessgo.Config.Listen.EnableHTTPS {
		c.TLSKeyfile = DefLessgo.Config.Listen.HTTPSKeyFile
		c.TLSCertfile = DefLessgo.Config.Listen.HTTPSCertFile
	}
	if len(listener) > 0 && listener[0] != nil {
		c.Listener = listener[0]
	}
	DefLessgo.Run(ns(c))
}

func rootGroup() {
	DefLessgo.Echo.Pre()
	DefLessgo.Echo.Suf()
}

func checkHooks(err error) {
	if err == nil {
		return
	}
	DefLessgo.Echo.Logger().Fatal("%v", err)
}
