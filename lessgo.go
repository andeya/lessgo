package lessgo

import (
	"github.com/lessgo/lessgo/engine"
	// "github.com/lessgo/lessgo/session"
)

type (
	Lessgo struct {
		*Echo
	}
	NewServer func(addr string) engine.Server
)

var (
	DefLessgo = &Lessgo{New()}
)

func Run(ns NewServer, addr string) {
	DefLessgo.Run(ns(addr))
}

func rootGroup() {
	DefLessgo.Echo.Pre()
	DefLessgo.Echo.Suf()
}
