package lessgo

import (
	"fmt"
	pathpkg "path"
	"sort"
	"strings"
	"sync"

	"github.com/go-xorm/xorm"

	"github.com/lessgo/lessgo/utils/uuid"
)

// 虚拟路由
type VirtRouter struct {
	Id         string   `json:"id" xorm:"not null pk VARCHAR(36)"` // UUID
	Pid        string   `json:"pid" xorm:"VARCHAR(36)"`            // 父节点id
	Type       int      `json:"type" xorm:"not null TINYINT(1)"`   // 操作类型: 根目录/路由分组/操作
	Name       string   `json:"name" xorm:"not null VARCHAR(500)"` // 名称(建议唯一)
	Enable     bool     `json:"enable" xorm:"not null TINYINT(1)"` // 是否启用当前路由节点
	Middleware []string `json:"middleware" xorm:"TEXT json"`       // 中间件 (允许运行时修改)
	Hid        string   `json:"hid" xorm:"not null VARCHAR(500)"`  // 虚拟操作VirtHandler.Id

	path     string        `xorm:"-"` // 路由匹配模式
	parent   *VirtRouter   `xorm:"-"` // 父节点
	children []*VirtRouter `xorm:"-"` // 子节点

	*VirtHandler `json:"virt_handler" xorm:"-"` // 虚拟操作
}

// 虚拟路由节点类型
const (
	ROOT int = iota
	GROUP
	HANDLER
)

var (
	// 数据库引擎
	lessgodb *xorm.Engine
	// 虚拟路由记录表，便于快速查找路由节点
	virtRouterMap  = map[string]*VirtRouter{}
	virtRouterLock sync.RWMutex
)

// 从数据库初始化虚拟路由
func initVirtRouterFromDB() (err error) {
	lessgodb, _ = GetDB("lessgo")
	if lessgodb == nil {
		return
	}
	err = lessgodb.Sync2(new(VirtRouter))
	if err != nil {
		return err
	}

	var dbInfo []*VirtRouter
	err = lessgodb.Find(&dbInfo)
	if err != nil {
		return fmt.Errorf("Failed to read virtRouter config: %v", err)
	}

	if len(dbInfo) == 0 {
		nodes := DefLessgo.VirtRouter.Progeny()
		_, err = lessgodb.Insert(&nodes)
		return
	}

	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	virtRouterMap = map[string]*VirtRouter{}

	for _, info := range dbInfo {
		info.VirtHandler = virtHandlerMap[info.Hid]
		virtRouterMap[info.Id] = info
	}
	for _, vr := range virtRouterMap {
		if vr.Type == ROOT {
			DefLessgo.VirtRouter = vr
		}
		parent := virtRouterMap[vr.Pid]
		if parent == nil {
			continue
		}
		vr.parent = parent
		parent.children = append(parent.children, vr)
	}
	return
}

// 创建虚拟路由根节点
func newRootVirtRouter() *VirtRouter {
	virtHandler := NewVirtHandler(nil, "/", nil, "", nil, nil)
	root := &VirtRouter{
		Id:          uuid.New().String(),
		Type:        ROOT,
		Name:        "Service Root",
		Enable:      true,
		VirtHandler: virtHandler,
		Middleware:  []string{},
		Hid:         virtHandler.Id(),
	}
	root.reset()
	return root
}

// 创建虚拟路由分组
func NewGroupVirtRouter(prefix, name string) *VirtRouter {
	virtHandler := NewVirtHandler(nil, prefix, nil, "", nil, nil)
	return &VirtRouter{
		Id:          uuid.New().String(),
		Type:        GROUP,
		Name:        name,
		Enable:      true,
		VirtHandler: virtHandler,
		Middleware:  []string{},
		Hid:         virtHandler.Id(),
	}
}

// 创建虚拟路由操作
func NewHandlerVirtRouter(name string, virtHandler *VirtHandler) *VirtRouter {
	if virtHandler == nil {
		virtHandler = &VirtHandler{}
	}
	return &VirtRouter{
		Id:          uuid.New().String(),
		Type:        HANDLER,
		Name:        name,
		Enable:      true,
		VirtHandler: virtHandler,
		Middleware:  []string{},
		Hid:         virtHandler.Id(),
	}
}

// 快速返回指定id对于的虚拟路由节点
func GetVirtRouter(id string) (*VirtRouter, bool) {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	vr, ok := virtRouterMap[id]
	return vr, ok
}

// 子孙虚拟路由节点列表
func (vr *VirtRouter) Progeny() []*VirtRouter {
	vrs := []*VirtRouter{vr}
	for _, novre := range vr.children {
		vrs = append(vrs, novre.Progeny()...)
	}
	return vrs
}

// 虚拟路由节点path
func (vr *VirtRouter) Path() string {
	return vr.path
}

// 返回改节点树上所有中间件的副本
func (vr *VirtRouter) AllMiddleware() []string {
	all := []string{}
	for _, m := range vr.Middleware {
		all = append(all, m)
	}
	for _, child := range vr.children {
		for _, m := range child.AllMiddleware() {
			all = append(all, m)
		}
	}
	return all
}

// 配置中间件
func (vr *VirtRouter) Use(middleware ...string) *VirtRouter {
	for _, m := range middleware {
		vr.Middleware = append(vr.Middleware, m)
	}
	return vr
}

// 重置中间件
func (vr *VirtRouter) ResetUse(middleware []string) (err error) {
	if middleware == nil {
		middleware = []string{}
	}
	if lessgodb == nil {
		goto label
	}
	_, err = lessgodb.Where("id=?", vr.Id).Cols("middleware").Update(&VirtRouter{Middleware: middleware})
	if err != nil {
		return
	}
label:
	vr.Middleware = middleware
	return
}

// 所有子节点的列表副本
func (vr *VirtRouter) Children() []*VirtRouter {
	cr := make([]*VirtRouter, len(vr.children))
	copy(cr, vr.children)
	return cr
}

// 添加子节点
func (vr *VirtRouter) AddChild(virtRouter *VirtRouter) (err error) {
	if virtRouter == nil {
		return fmt.Errorf("Can not add an empty node.")
	}
	if virtRouter.Type == ROOT {
		return fmt.Errorf("Can not add an root node.")
	}
	virtRouter.Pid = vr.Id
	virtRouter.parent = vr
	if lessgodb == nil {
		goto label
	}
	_, err = lessgodb.Insert(virtRouter)
	if err != nil {
		return
	}
label:
	vr.children = append(vr.children, virtRouter)
	virtRouter.reset()
	addVirtRouter(virtRouter)
	return nil
}

// 删除子节点
func (vr *VirtRouter) DelChild(virtRouter *VirtRouter) (err error) {
	if virtRouter == nil {
		return fmt.Errorf("Can not delete an empty node.")
	}
	var session *xorm.Session
	nodes := virtRouter.Progeny()
	if lessgodb == nil {
		goto label
	}
	session = lessgodb.NewSession()
	defer session.Close()
	err = session.Begin()
	if err != nil {
		return err
	}
	for _, v := range nodes {
		_, err = session.Delete(v)
		if err != nil {
			session.Rollback()
			return
		}
	}
	err = session.Commit()
	if err != nil {
		return
	}
label:
	var exist bool
	for i, child := range vr.children {
		if child == virtRouter {
			vr.children = append(vr.children[:i], vr.children[i+1:]...)
			exist = true
			break
		}
	}
	if exist {
		for _, node := range nodes {
			delVirtRouter(node)
		}
		return nil
	}
	return fmt.Errorf("node %v does not have child node: %v.", vr.Name, virtRouter.Name)
}

// 虚拟路由节点的父节点
func (vr *VirtRouter) Parent() *VirtRouter {
	return vr.parent
}

// 删除自身
func (vr *VirtRouter) Delete() error {
	if vr.parent != nil {
		vr.parent.DelChild(vr)
		return nil
	}
	return fmt.Errorf("Can not delete the root node.")
}

// 根据父节点重置虚拟路由节点自身及其子节点path
func (vr *VirtRouter) reset() {
	var parentPath = "/"
	if vr.parent != nil {
		parentPath = vr.parent.path
	}
	var prefix string
	if vr.VirtHandler != nil {
		prefix = vr.VirtHandler.prefix
	}
	vr.path = pathpkg.Clean(pathpkg.Join("/", parentPath, prefix))
	for _, child := range vr.children {
		child.reset()
	}
}

func addVirtRouter(vr *VirtRouter) bool {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	if _, ok := virtRouterMap[vr.Id]; ok {
		return false
	}
	virtRouterMap[vr.Id] = vr
	return true
}

func resetVirtRouter(oldId string, vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, oldId)
	virtRouterMap[vr.Id] = vr
}

func delVirtRouter(vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, vr.Id)
}

func cleanVirtRouterMap() {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	virtRouterMap = map[string]*VirtRouter{}
}

/*
 * 注册真实路由
 */
func (vr *VirtRouter) route(group *Group) {
	if !vr.Enable {
		return
	}
	mws := getMiddlewares(vr.Middleware)
	prefix := vr.VirtHandler.Prefix()
	prefix2 := pathpkg.Join("/", strings.TrimSuffix(vr.VirtHandler.PrefixPath(), "/index"), vr.VirtHandler.PrefixParam())
	hasIndex := prefix2 != prefix
	switch vr.Type {
	case GROUP:
		var childGroup *Group
		if hasIndex {
			// "/index"分组会被默认为"/"
			childGroup = group.Group(prefix2, mws...)
		} else {
			childGroup = group.Group(prefix, mws...)
		}
		for _, child := range vr.Children() {
			child.route(childGroup)
		}
	case HANDLER:
		methods := vr.VirtHandler.Methods()
		handler := getHandlerMap(vr.VirtHandler.Id())
		if hasIndex {
			group.Match(methods, prefix2, handler, mws...)
		}
		group.Match(methods, prefix, handler, mws...)
	}
}

func route(methods []string, prefix, name string, descHandlerOrhandler interface{}, middleware []string) *VirtRouter {
	sort.Strings(methods)
	var (
		handler     HandlerFunc
		description string
		params      []Param
		produces    []string
	)
	switch h := descHandlerOrhandler.(type) {
	case HandlerFunc:
		handler = h
	case func(Context) error:
		handler = HandlerFunc(h)
	case DescHandler:
		handler = h.Handler
		description = h.Desc
		params = h.Params
		produces = h.Produces
	case *DescHandler:
		handler = h.Handler
		description = h.Desc
		params = h.Params
		produces = h.Produces
	}
	// 生成VirtHandler
	virtHandler := NewVirtHandler(handler, prefix, methods, description, produces, params)
	// 生成虚拟路由操作
	vth := NewHandlerVirtRouter(name, virtHandler)
	err := vth.ResetUse(middleware)
	if err != nil {
		Logger().Error("%v", err)
	}
	return vth
}

func registerVirtRouter() {
	if err := middlewareCheck(DefLessgo.virtBefore); err != nil {
		Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err := middlewareCheck(DefLessgo.virtAfter); err != nil {
		Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err := middlewareCheck(DefLessgo.VirtRouter.AllMiddleware()); err != nil {
		Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}

	// 从虚拟路由创建真实路由
	DefLessgo.app.router = NewRouter(DefLessgo.app)
	DefLessgo.app.middleware = []MiddlewareFunc{DefLessgo.app.router.Process}
	DefLessgo.app.head = DefLessgo.app.pristineHead
	DefLessgo.app.BeforeUse(getMiddlewares(DefLessgo.virtBefore)...)
	DefLessgo.app.AfterUse(getMiddlewares(DefLessgo.virtAfter)...)
	group := DefLessgo.app.Group(
		DefLessgo.VirtRouter.Prefix(),
		getMiddlewares(DefLessgo.VirtRouter.Middleware)...,
	)
	for _, child := range DefLessgo.VirtRouter.Children() {
		child.route(group)
	}
}
