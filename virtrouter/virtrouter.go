package virtrouter

import (
	"fmt"
	"path"
	"sync"
)

// 虚拟路由
type VirtRouter struct {
	url         string        // 访问链接(=parent.url+prefix+VirtHandler.suffix)
	typ         int           // 操作类型: 根目录/路由分组/操作
	prefix      string        // 路由节点的url前缀路径(允许运行时修改)
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

// 快速返回指定url对于的虚拟路由节点
func GetVirtRouter(u string) (*VirtRouter, bool) {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	vr, ok := virtRouterMap[u]
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
			children[i] = c.url
		}
		RegSerialRouter(&SerialRouter{
			Url:           v.url,
			Type:          v.typ,
			Prefix:        v.prefix,
			Name:          v.name,
			Children:      children,
			Enable:        v.enable,
			Middleware:    v.middleware,
			VirtHandlerId: v.virtHandler.id,
		})
	}
	return SerialRouterMap()
}

func NewVirtRouter(typ int, prefix, name string, virtHandler *VirtHandler) *VirtRouter {
	if virtHandler == nil {
		virtHandler = &VirtHandler{}
	}
	return &VirtRouter{
		typ:         typ,
		prefix:      path.Clean(path.Join("/", prefix)),
		name:        name,
		enable:      true,
		virtHandler: virtHandler,
		children:    []*VirtRouter{},
	}
}

// 创建虚拟路由根节点
func NewRootVirtRouter() (*VirtRouter, error) {
	root := &VirtRouter{
		typ:         ROOT,
		prefix:      "/",
		name:        "根路径",
		enable:      true,
		virtHandler: &VirtHandler{},
		children:    []*VirtRouter{},
	}
	if !addVirtRouter(root) {
		return nil, fmt.Errorf("不可重复创建根节点")
	}
	root.resetUrls()
	return root, nil
}

// 创建虚拟路由子节点
func (vr *VirtRouter) NewChild(typ int, prefix, name string, virtHandler *VirtHandler) bool {
	switch typ {
	case GROUP, HANDLER:
	default:
		return false
	}
	if virtHandler == nil {
		virtHandler = &VirtHandler{}
	}
	child := &VirtRouter{
		typ:         typ,
		prefix:      path.Clean(path.Join("/", prefix)),
		name:        name,
		enable:      true,
		virtHandler: virtHandler,
		children:    []*VirtRouter{},
	}
	child.parent = vr
	child.resetUrls()
	if !addVirtRouter(child) {
		return false
	}
	vr.children = append(vr.children, child)
	return true
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

// 虚拟路由节点url前缀
func (vr *VirtRouter) Prefix() string {
	return vr.prefix
}

// 设置虚拟路由节点url前缀及其url
func (vr *VirtRouter) SetPrefix(prefix string) {
	vr.prefix = prefix
	vr.resetUrls()
}

// 虚拟路由节点url
func (vr *VirtRouter) Url() string {
	return vr.url
}

// 虚拟路由节点url当前分路径
func (vr *VirtRouter) SubUrl() string {
	return path.Join(vr.prefix, vr.virtHandler.suffix)
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

// 配置中间件
func (vr *VirtRouter) Use(middleware ...string) *VirtRouter {
	for _, m := range middleware {
		vr.middleware = append(vr.middleware, m)
	}
	return vr
}

// 重置中间件
func (vr *VirtRouter) ResetUse(middleware ...string) *VirtRouter {
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
	virtRouter.resetUrls()
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
	return fmt.Errorf("当前虚拟路由节点不存在子节点 %v", virtRouter.url)
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
	vr.resetUrls()
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

// 根据父节点重置虚拟路由节点自身及其子节点url
func (vr *VirtRouter) resetUrls() {
	oldUrl := vr.url
	var parentUrl = "/"
	if vr.parent != nil {
		parentUrl = vr.parent.url
	}
	var suffix string
	if vr.virtHandler != nil {
		suffix = vr.virtHandler.suffix
	}
	vr.url = path.Clean(path.Join("/", parentUrl, vr.prefix, suffix))
	resetVirtRouter(oldUrl, vr)

	for _, child := range vr.children {
		child.resetUrls()
	}
}

func addVirtRouter(vr *VirtRouter) bool {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	if _, ok := virtRouterMap[vr.url]; ok {
		return false
	}
	virtRouterMap[vr.url] = vr
	return true
}

func resetVirtRouter(oldUrl string, vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, oldUrl)
	virtRouterMap[vr.url] = vr
}

func delVirtRouter(vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, vr.url)
}

func cleanVirtRouterMap() {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	virtRouterMap = map[string]*VirtRouter{}
}
