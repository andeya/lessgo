package lessgo

import (
	"path"
	"reflect"
	"runtime"
	"sort"
	"strings"

	"github.com/lessgo/lessgo/utils"
)

type (
	DynaRouter struct {
		Id          string
		Url         string
		Type        int
		Prefix      string // 允许动态修改
		Name        string // 允许动态修改
		Description string // 允许动态修改
		Methods     []string
		Param       string
		Handler     string
		Middlewares []string // 允许动态修改
		ParentUrl   string   // 允许动态指定父节点
		Parent      *DynaRouter
		Children    []*DynaRouter
	}
)

const (
	ROOT int = iota - 1
	GROUP
	HANDLER
)

var (
	HandlerFuncMap = map[string]HandlerFunc{}
	MiddlewareMap  = map[string]Middleware{}
	dynaRouterMap  = map[string]*DynaRouter{}
	DefDynaRouter  = &DynaRouter{
		Prefix:      "/",
		Type:        ROOT,
		Middlewares: []string{},
		Children:    []*DynaRouter{},
	}
)

func (d *DynaRouter) Tree() []*DynaRouter {
	rs := []*DynaRouter{d}
	for _, node := range d.Children {
		rs = append(rs, node.Tree()...)
	}
	return rs
}

func (d *DynaRouter) SetMiddlewares(middlewares []string) {
	d.Middlewares = middlewares
}

func (d *DynaRouter) init() {
	d.setUrl()
	d.setId()
	for _, node := range d.Children {
		node.init()
	}
	dynaRouterMap[d.Id] = d
}

func (d *DynaRouter) setUrl() {
	var u = path.Join(d.Prefix, d.Param)
	d.ParentUrl = ""
	if d.Parent != nil {
		d.Parent.setUrl()
		d.ParentUrl = d.Parent.Url
		u = path.Join(d.Parent.Url, u)
	}
	d.Url = u
}

func (d *DynaRouter) setId() {
	d.Id = utils.MakeMd5(d.Url + strings.Join(d.Methods, ""))
}

// 返回路由列表
func DynaRouterTree() []*DynaRouter {
	return DefDynaRouter.Tree()
}

// 设置或添加路由
func SetRouter(parentId string, node *DynaRouter) {
	node.Parent = dynaRouterMap[parentId]
	node.init()
	dynaRouterMap[node.Id] = node
	for i, child := range node.Parent.Children {
		if child.Id == node.Id {
			node.Parent.Children[i] = node
			return
		}
	}
	node.Parent.Children = append(node.Parent.Children, node)
}

// 移除路由
func DelRouter(nodeId string) {
	parent := dynaRouterMap[nodeId].Parent
	for i, child := range parent.Children {
		if child.Id == nodeId {
			parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
			break
		}
	}
	delete(dynaRouterMap, nodeId)
}

// 重建真实路由
func ResetRealRoute() {
	DefLessgo.Echo.lock.Lock()
	defer DefLessgo.Echo.lock.Unlock()
	DefLessgo.Echo.router = NewRouter(DefLessgo.Echo)
	DefLessgo.Echo.middleware = []Middleware{DefLessgo.Echo.router}
	DefLessgo.Echo.head = HandlerFunc(func(c Context) error {
		return c.Handle(c)
	})
	DefLessgo.Echo.pristineHead = DefLessgo.Echo.head
	DefLessgo.Echo.chainMiddleware()
	var group *Group
	for _, d := range DynaRouterTree() {
		var mws = make([]Middleware, len(d.Middlewares))
		for i, mw := range d.Middlewares {
			mws[i] = MiddlewareMap[mw]
		}

		switch d.Type {
		case ROOT:
			DefLessgo.Echo.Use(mws...)
		case GROUP:
			if group == nil {
				group = DefLessgo.Echo.Group(d.Prefix, mws...)
				break
			}
			group = group.Group(d.Prefix, mws...)
		case HANDLER:
			if group == nil {
				DefLessgo.Echo.Match(d.Methods, path.Join(d.Prefix, d.Param), HandlerFuncMap[d.Handler], mws...)
				break
			}
			group.Match(d.Methods, path.Join(d.Prefix, d.Param), HandlerFuncMap[d.Handler], mws...)
		}
	}
}

// 从根路由开始配置路由
func RootRouter(node ...*DynaRouter) {
	DefDynaRouter.Children = append(DefDynaRouter.Children, node...)
	for _, r := range node {
		r.Parent = DefDynaRouter
	}
	DefDynaRouter.init()
}

// 配置路由分组
func SubRouter(prefix, name, description string, node ...*DynaRouter) *DynaRouter {
	p := &DynaRouter{
		Prefix:      prefix,
		Type:        GROUP,
		Name:        name,
		Description: description,
		Middlewares: []string{},
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
	sort.Strings(methods)
	hUri := handleWareUri(handler)
	HandlerFuncMap[hUri] = handler
	if len(param) == 0 {
		param = append(param, "")
	}
	ns := strings.Split(hUri, ".")
	n := strings.TrimSuffix(ns[len(ns)-1], "Handle")
	prefix := "/" + utils.SnakeString(n)
	return &DynaRouter{
		Prefix:      prefix,
		Type:        HANDLER,
		Name:        name,
		Description: description,
		Methods:     methods,
		Param:       param[0],
		Handler:     hUri,
		Middlewares: []string{},
	}
}

func handleWareUri(hw interface{}) string {
	t := reflect.ValueOf(hw).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(hw).Pointer()).Name()
	}
	return t.String()
}
