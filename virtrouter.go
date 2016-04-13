package lessgo

import (
	"fmt"
	"path"
	"sort"
	"strings"
	"sync"
)

// 虚拟路由
type VirtRouter struct {
	id          string        // parent.id+VirtHandler.prefix+[METHOD1]+[METHOD...]
	typ         int           // 操作类型: 根目录/路由分组/操作
	name        string        // 名称(建议唯一)
	parent      *VirtRouter   // 父节点
	children    []*VirtRouter // 子节点列表
	enable      bool          // 是否启用当前路由节点
	middleware  []string      // 中间件 (允许运行时修改)
	virtHandler *VirtHandler  // 虚拟操作
}

// 虚拟路由节点类型
const (
	ROOT int = iota
	GROUP
	HANDLER
)

// 虚拟路由记录表，便于快速查找路由节点
var (
	virtRouterMap  = map[string]*VirtRouter{}
	virtRouterLock sync.RWMutex
)

// 快速返回指定id对于的虚拟路由节点
func GetVirtRouter(id string) (*VirtRouter, bool) {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	vr, ok := virtRouterMap[id]
	return vr, ok
}

// 返回虚拟路由记录表的副本
func VirtRouterMap() map[string]*VirtRouter {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	vrs := make(map[string]*VirtRouter, len(virtRouterMap))
	for k, v := range virtRouterMap {
		vrs[k] = v
	}
	return vrs
}

// 序列化虚拟路由并返回副本
func ToSerialRouter() map[string]*SerialRouter {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	// 清空serialRouterMap
	cleanSerialRouterMap()
	// 执行序列化到serialRouterMap
	for _, v := range virtRouterMap {
		children := make([]string, len(v.children))
		for i, c := range v.children {
			children[i] = c.id
		}
		RegSerialRouter(&SerialRouter{
			Id:            v.id,
			Type:          v.typ,
			Name:          v.name,
			Children:      children,
			Enable:        v.enable,
			Middleware:    v.middleware,
			VirtHandlerId: v.virtHandler.id,
		})
	}
	return SerialRouterMap()
}

// 创建虚拟路由根节点
func NewVirtRouterRoot() (*VirtRouter, error) {
	root := &VirtRouter{
		typ:         ROOT,
		name:        "根路径",
		enable:      true,
		virtHandler: &VirtHandler{prefix: "/"},
		children:    []*VirtRouter{},
	}
	if !addVirtRouter(root) {
		return nil, fmt.Errorf("不可重复创建根节点")
	}
	root.resetIds()
	return root, nil
}

// 创建虚拟路由分组
func NewVirtRouterGroup(prefix, name string) *VirtRouter {
	prefix, prefixPath, prefixParam := creatPrefix(prefix)
	return &VirtRouter{
		typ:    GROUP,
		name:   name,
		enable: true,
		virtHandler: &VirtHandler{
			prefix:      prefix,
			prefixPath:  prefixPath,
			prefixParam: prefixParam,
		},
		children: []*VirtRouter{},
	}
}

// 创建虚拟路由操作
func NewVirtRouterHandler(name string, virtHandler *VirtHandler) *VirtRouter {
	if virtHandler == nil {
		virtHandler = &VirtHandler{}
	}
	return &VirtRouter{
		typ:         HANDLER,
		name:        name,
		enable:      true,
		virtHandler: virtHandler,
		children:    []*VirtRouter{},
	}
}

// 子孙虚拟路由节点列表
func (vr *VirtRouter) Progeny() []*VirtRouter {
	vrs := []*VirtRouter{vr}
	for _, novre := range vr.children {
		vrs = append(vrs, novre.Progeny()...)
	}
	return vrs
}

// 虚拟路由节点类型值
func (vr *VirtRouter) Type() int {
	return vr.typ
}

// 虚拟路由节点名称
func (vr *VirtRouter) Name() string {
	return vr.name
}

// 设置虚拟路由节点名称
func (vr *VirtRouter) SetName(name string) {
	vr.name = name
}

// 虚拟路由节点id
func (vr *VirtRouter) Id() string {
	return vr.id
}

// 虚拟路由节点操作
func (vr *VirtRouter) VirtHandler() *VirtHandler {
	return vr.virtHandler
}

// 返回中间件的副本
func (vr *VirtRouter) Middleware() []string {
	m := make([]string, len(vr.middleware))
	copy(m, vr.middleware)
	return m
}

// 返回改节点树上所有中间件的副本
func (vr *VirtRouter) AllMiddleware() []string {
	all := []string{}
	for _, m := range vr.middleware {
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
		vr.middleware = append(vr.middleware, m)
	}
	return vr
}

// 重置中间件
func (vr *VirtRouter) ResetUse(middleware []string) *VirtRouter {
	vr.middleware = middleware
	return vr
}

// 虚拟路由节点的启用状态
func (vr *VirtRouter) Enable() bool {
	return vr.enable
}

// 配置启用状态，默认为启用
func (vr *VirtRouter) SetEnable(enable bool) *VirtRouter {
	vr.enable = enable
	for _, child := range vr.children {
		child.SetEnable(enable)
	}
	return vr
}

// 所有子节点的列表副本
func (vr *VirtRouter) Children() []*VirtRouter {
	cr := make([]*VirtRouter, len(vr.children))
	copy(cr, vr.children)
	return cr
}

// 添加子节点
func (vr *VirtRouter) AddChild(virtRouter *VirtRouter) error {
	if virtRouter == nil {
		return fmt.Errorf("不可添加空的子节点")
	}
	virtRouter.parent = vr
	vr.children = append(vr.children, virtRouter)
	virtRouter.resetIds()
	return nil
}

// 添加多个子节点
func (vr *VirtRouter) AddChildren(virtRouters []*VirtRouter) *VirtRouter {
	for _, child := range virtRouters {
		if err := vr.AddChild(child); err != nil {
			fmt.Println(err)
		}
	}
	return vr
}

// 删除子节点
func (vr *VirtRouter) DelChild(virtRouter *VirtRouter) error {
	if virtRouter == nil {
		return fmt.Errorf("欲删除的虚拟路由节点不能为nil")
	}
	for i, child := range vr.children {
		if child == virtRouter {
			vr.children = append(vr.children[:i], vr.children[i+1:]...)
			delVirtRouter(virtRouter)
			return nil
		}
	}
	return fmt.Errorf("当前虚拟路由节点不存在子节点 %v", virtRouter.id)
}

// 虚拟路由节点的父节点
func (vr *VirtRouter) Parent() *VirtRouter {
	return vr.parent
}

// 设置虚拟路由节点的父节点
func (vr *VirtRouter) SetParent(virtRouter *VirtRouter) error {
	if virtRouter == nil {
		return fmt.Errorf("不可将空节点设置为父节点")
	}
	vr.Delete()
	vr.parent = virtRouter
	virtRouter.children = append(virtRouter.children, vr)
	vr.resetIds()
	return nil
}

// 删除自身
func (vr *VirtRouter) Delete() error {
	if vr.parent != nil {
		vr.parent.DelChild(vr)
		return nil
	}
	return fmt.Errorf("不能删除虚拟路由根节点")
}

// 根据父节点重置虚拟路由节点自身及其子节点id
func (vr *VirtRouter) resetIds() {
	oldId := vr.id
	var parentId = "/"
	if vr.parent != nil {
		parentId = vr.parent.id
	}
	var prefix string
	if vr.virtHandler != nil {
		prefix = vr.virtHandler.prefix
	}
	vr.id = path.Clean(path.Join("/", parentId, prefix))
	for _, m := range vr.virtHandler.methods {
		vr.id += "[" + m + "]"
	}
	resetVirtRouter(oldId, vr)
	for _, child := range vr.children {
		child.resetIds()
	}
}

func addVirtRouter(vr *VirtRouter) bool {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	if _, ok := virtRouterMap[vr.id]; ok {
		return false
	}
	virtRouterMap[vr.id] = vr
	return true
}

func resetVirtRouter(oldId string, vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, oldId)
	virtRouterMap[vr.id] = vr
}

func delVirtRouter(vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, vr.id)
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
	if !vr.Enable() {
		return
	}
	mws := getMiddlewares(vr.Middleware())
	prefix := vr.VirtHandler().Prefix()
	prefix2 := path.Join("/", strings.TrimSuffix(vr.VirtHandler().PrefixPath(), "/index"), vr.VirtHandler().PrefixParam())
	hasIndex := prefix2 != prefix
	switch vr.Type() {
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
		methods := vr.VirtHandler().Methods()
		handler := getHandlerMap(vr.VirtHandler().Id())
		if hasIndex {
			group.Match(methods, prefix2, handler, mws...)
		}
		group.Match(methods, prefix, handler, mws...)
	}
}

func route(methods []string, prefix, name string, descHandlerOrhandler interface{}, middleware []string) *VirtRouter {
	sort.Strings(methods)
	var (
		handler                       HandlerFunc
		description, success, failure string
		param                         map[string]string
	)
	switch h := descHandlerOrhandler.(type) {
	case HandlerFunc:
		handler = h
	case func(Context) error:
		handler = HandlerFunc(h)
	case DescHandler:
		handler = h.Handler
		description = h.Desc
		param = h.Param
	case *DescHandler:
		handler = h.Handler
		description = h.Desc
		param = h.Param
	}
	// 生成VirtHandler
	virtHandler := NewVirtHandler(handler, prefix, methods, description, success, failure, param)
	// 生成虚拟路由操作
	return NewVirtRouterHandler(name, virtHandler).ResetUse(middleware)
}
