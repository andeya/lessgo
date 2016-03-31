package lessgo

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/lessgo/lessgo/utils"
)

type (
	DynaRouter struct {
		Prefix          string
		Name            string
		Description     string
		Methods         []string
		Param           string
		HandlerFuncName string
		Middlewares     []Middleware
		Parent          *DynaRouter
		Children        []*DynaRouter
	}
	RouterFunc func(name, description string, handler HandlerFunc, param ...string)
)

var (
	DefDynaRouter = &DynaRouter{
		Prefix:      "/",
		Middlewares: []Middleware{},
		Children:    []*DynaRouter{},
	}
	HandlerFuncMap = map[string]HandlerFunc{}
)

func (d *DynaRouter) Url() string {
	var u = path.Join(d.Prefix, d.Param)
	if d.Parent != nil {
		u = path.Join(d.Parent.Url(), u)
	}
	return u
}

func (d *DynaRouter) Tree() []*DynaRouter {
	rs := []*DynaRouter{d}
	for _, node := range d.Children {
		rs = append(rs, node.Tree()...)
	}
	return rs
}

func DynaRouterTree() []*DynaRouter {
	return DefDynaRouter.Tree()
}

func RootRouter(node ...*DynaRouter) {
	DefDynaRouter.Children = node
	for _, r := range node {
		r.Parent = DefDynaRouter
	}
}

func SubRouter(prefix, name, description string, node ...*DynaRouter) *DynaRouter {
	p := &DynaRouter{
		Prefix:      prefix,
		Name:        name,
		Description: description,
		Middlewares: []Middleware{},
		Children:    node,
	}
	for _, r := range node {
		r.Parent = p
	}
	return p
}

func Get(name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	return route([]string{GET}, name, description, handler, param...)
}
func Head(name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	return route([]string{HEAD}, name, description, handler, param...)
}
func Options(name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	return route([]string{OPTIONS}, name, description, handler, param...)
}
func Patch(name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	return route([]string{PATCH}, name, description, handler, param...)
}
func Post(name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	return route([]string{POST}, name, description, handler, param...)
}
func Put(name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	return route([]string{PUT}, name, description, handler, param...)
}
func Trace(name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	return route([]string{TRACE}, name, description, handler, param...)
}
func Any(name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	return route([]string{CONNECT, DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT, TRACE}, name, description, handler, param...)
}
func Match(methods []string, name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	if len(methods) == 0 {
		DefLessgo.logger.Error("The method can not be empty: %v", name)
	}
	return route(methods, name, description, handler, param...)
}

func route(methods []string, name, description string, handler HandlerFunc, param ...string) *DynaRouter {
	hname := handlerName(handler)
	HandlerFuncMap[hname] = handler
	if len(param) == 0 {
		param = append(param, "")
	}
	ns := strings.Split(hname, ".")
	n := strings.TrimSuffix(ns[len(ns)-1], "Handle")
	prefix := "/" + utils.SnakeString(n)
	return &DynaRouter{
		Prefix:          prefix,
		Name:            name,
		Description:     description,
		Methods:         methods,
		Param:           param[0],
		HandlerFuncName: handlerName(handler),
		Middlewares:     []Middleware{},
	}
}

func DynaRouterToJson() string {
	b, _ := json.Marshal(DefDynaRouter)
	return string(b)
}
