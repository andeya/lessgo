package lessgo

import (
	"github.com/lessgo/lessgo/engine"
	// "github.com/lessgo/lessgo/session"
)

type (
	NewServer func(addr string) engine.Server
)

var (
	DefLessgo = New()
)

func Run(ns NewServer, addr string) {
	DefLessgo.Run(ns(addr))
}
