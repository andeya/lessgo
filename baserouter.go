package lessgo

import (
	"sync"
)

type baseRouter struct {
	Path       string
	Methods    []string
	Handler    HandlerFunc
	Middleware []MiddlewareFunc
}

type staticBaseRouter struct {
	Prefix     string
	Root       string
	Middleware []MiddlewareFunc
}

type fileBaseRouter struct {
	Path       string
	File       string
	Middleware []MiddlewareFunc
}

var (
	baseRouterMap       = map[string]*baseRouter{}
	staticBaseRouterMap = map[string]*staticBaseRouter{}
	fileBaseRouterMap   = map[string]*fileBaseRouter{}
	baseRouterLock      sync.Mutex
)

func registerBaseRouter() {
	baseRouterLock.Lock()
	for _, br := range baseRouterMap {
		DefLessgo.Echo.Match(br.Methods, br.Path, br.Handler, br.Middleware...)
	}
	for _, sbr := range staticBaseRouterMap {
		DefLessgo.Echo.Static(sbr.Prefix, sbr.Root, sbr.Middleware...)
	}
	for _, fbr := range fileBaseRouterMap {
		DefLessgo.Echo.File(fbr.Path, fbr.File, fbr.Middleware...)
	}
	baseRouterLock.Unlock()
}
