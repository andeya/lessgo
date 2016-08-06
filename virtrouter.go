package lessgo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	pathpkg "path"
	"sort"
	"strings"
	"sync"

	"github.com/lessgo/lessgo/utils"
	"github.com/lessgo/lessgoext/uuid"
)

// 单独注册的静态文件虚拟路由(无法在Root()下使用，暂不支持运行时修改)
type VirtFile struct {
	Path        string
	File        string
	Middlewares []*MiddlewareConfig
}

// 从单独静态文件虚拟路由注册真实路由
func (this *VirtFile) route() {
	app.file(this.Path, this.File, getMiddlewareFuncs(this.Middlewares)...)
}

// 单独注册的静态目录虚拟路由(无法在Root()下使用，暂不支持运行时修改)
type VirtStatic struct {
	Prefix      string
	Root        string
	Middlewares []*MiddlewareConfig
}

// 从单独静态目录虚拟路由注册真实路由
func (this *VirtStatic) route() {
	app.static(this.Prefix, this.Root, getMiddlewareFuncs(this.Middlewares)...)
}

// 虚拟路由(在Root()下使用，支持运行时修改)
type VirtRouter struct {
	Id          string              `json:"id""`         // UUID
	Type        int                 `json:"type""`       // 操作类型: 根目录/路由分组/操作
	Prefix      string              `json:"prefix"`      // 路由节点的url前缀(不含参数)
	Middlewares []*MiddlewareConfig `json:"middlewares"` // 中间件列表 (允许运行时修改)
	Enable      bool                `json:"enable"`      // 是否启用当前路由节点
	Dynamic     bool                `json:"dynamic"`     // 是否动态追加的节点
	Hid         string              `json:"hid"`         // 操作ApiHandler.id
	Children    virtRouterSlice     `json:"children"`    // 子节点
	Parent      *VirtRouter         `json:"-"`           // 父节点

	path       string      `json:"-"` // 路由匹配模式
	suffix     string      `json:"-"` // 路由匹配模式path参数后缀
	apiHandler *ApiHandler `json:"-"` // 操作
	params     []Param     `json:"-"` // 所有参数的列表，含前面节点的中间件
}

// 虚拟路由节点类型
const (
	ROOT int = iota
	GROUP
	HANDLER
)

var (
	// 虚拟路由记录表，便于快速查找路由节点
	virtRouterMap  = map[string]*VirtRouter{}
	virtRouterLock sync.RWMutex

	// 非动态节点不可修改
	notDynamicError = fmt.Errorf("The specified node is not dynamic, and therefore can not be modified.")
)

// 获取操作的请求方法列表(已排序)
func (vr *VirtRouter) Methods() []string {
	return vr.apiHandler.Methods()
}

// 操作的描述
func (vr *VirtRouter) Description() string {
	return vr.apiHandler.Desc
}

// 节点所有参数的说明列表
func (vr *VirtRouter) Params() []Param {
	return vr.params
}

// 操作的返回结果说明列表
func (vr *VirtRouter) HTTP200() []Result {
	return vr.apiHandler.HTTP200
}

// 子孙虚拟路由节点列表
func (vr *VirtRouter) Progeny() []*VirtRouter {
	vrs := []*VirtRouter{vr}
	for _, novre := range vr.Children {
		vrs = append(vrs, novre.Progeny()...)
	}
	return vrs
}

// 虚拟路由节点path
func (vr *VirtRouter) Path() string {
	return vr.path
}

// 设置虚拟路由节点url前缀
func (vr *VirtRouter) SetPrefix(prefix string) (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	_orgin := vr.Prefix
	vr.Prefix = prefix
	vr.reset()
	err = saveVirtRouterConfig()
	if err != nil {
		// 数据回滚
		vr.Prefix = _orgin
		vr.reset()
	}
	return
}

// 操作的参数匹配模式
func (vr *VirtRouter) Suffix() string {
	return vr.suffix
}

// 启用/禁用虚拟路由节点
func (vr *VirtRouter) SetEnable(able bool) (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	_orgin := vr.Enable
	vr.Enable = able
	err = saveVirtRouterConfig()
	if err != nil {
		// 数据回滚
		vr.Enable = _orgin
	}
	return
}

// 配置中间件(仅在源码中使用)，
// 因为源码路由书写格式需要，仅允许返回*VirtRouter一个参数，
// 而动态配置时需要有error反馈因此，该方法仅限源码中使用。
func (vr *VirtRouter) Use(middlewares ...*ApiMiddleware) *VirtRouter {
	if vr.Dynamic {
		Log.Error("Specified node is dynamic, please use ResetUse(middlewares []string) (err error).")
		return vr
	}
	l := len(middlewares)
	if l == 0 {
		return vr
	}
	_l := len(vr.Middlewares)
	ms := make([]*MiddlewareConfig, _l+l)
	copy(ms, vr.Middlewares)
	for i, m := range middlewares {
		m.init()
		ms[i+_l] = m.NewMiddlewareConfig()
	}
	vr.Middlewares = ms
	return vr
}

// 重置中间件
func (vr *VirtRouter) ResetUse(middlewares []*MiddlewareConfig) (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	if middlewares == nil {
		middlewares = []*MiddlewareConfig{}
	}
	_orgin := vr.Middlewares
	vr.Middlewares = middlewares
	err = saveVirtRouterConfig()
	if err != nil {
		// 数据回滚
		vr.Middlewares = _orgin
	}
	return
}

// 为节点更换操作
func (vr *VirtRouter) SetApiHandler(hid string) (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	vh := getApiHandler(hid)
	if vh == nil {
		return fmt.Errorf("Specified ApiHandler does not exist.")
	}

	_hid := vr.Hid
	_vh := vr.apiHandler
	vr.Hid = hid
	vr.apiHandler = vh
	vr.reset()
	err = saveVirtRouterConfig()
	if err != nil {
		// 数据回滚
		vr.Hid = _hid
		vr.apiHandler = _vh
		vr.reset()
	}
	return
}

// 添加子节点(仅限动态配置时使用)
func (vr *VirtRouter) AddChild(virtRouter *VirtRouter) (err error) {
	if !virtRouter.Dynamic {
		return notDynamicError
	}
	return vr.addChild(virtRouter)
}

// 删除子节点(仅限动态配置时使用)
func (vr *VirtRouter) DelChild(virtRouter *VirtRouter) (err error) {
	if !virtRouter.Dynamic {
		return notDynamicError
	}
	return vr.delChild(virtRouter)
}

// 删除自身(仅限动态配置时使用)
func (vr *VirtRouter) Delete() (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	return vr.delete()
}

// 添加子节点
func (vr *VirtRouter) addChild(virtRouter *VirtRouter) (err error) {
	if virtRouter == nil {
		return fmt.Errorf("Can not add an empty node.")
	}
	if virtRouter.Type == ROOT {
		return fmt.Errorf("Can not add an root node.")
	}

	virtRouter.Parent = vr
	children := vr.Children
	vr.Children = append(vr.Children, virtRouter)
	virtRouter.reset()
	err = saveVirtRouterConfig()
	if err != nil {
		// 数据回滚
		vr.Children = children
		virtRouter.reset()
	} else {
		addVirtRouter(virtRouter)
	}
	return
}

// 删除自身
func (vr *VirtRouter) delete() error {
	if vr.Parent != nil {
		vr.Parent.delChild(vr)
		return nil
	}
	return fmt.Errorf("Can not delete the root node.")
}

// 删除子节点
func (vr *VirtRouter) delChild(virtRouter *VirtRouter) (err error) {
	if virtRouter == nil {
		return fmt.Errorf("Can not delete an empty node.")
	}
	var exist bool
	for i, child := range vr.Children {
		if child == virtRouter {
			children := make([]*VirtRouter, len(vr.Children))
			copy(children, vr.Children)
			vr.Children = append(vr.Children[:i], vr.Children[i+1:]...)
			err = saveVirtRouterConfig()
			if err != nil {
				// 数据回滚
				vr.Children = children
				return
			} else {
				exist = true
			}
			break
		}
	}
	if exist {
		for _, node := range virtRouter.Progeny() {
			delVirtRouter(node)
		}
		return nil
	}
	return fmt.Errorf("node %v does not have child node: %v.", vr.Description(), virtRouter.Description())
}

// 对从配置文件读来的路由进行部分字段的初始化
func (vr *VirtRouter) initFromConfig() {
	// 获取操作
	vr.apiHandler = getApiHandler(vr.Hid)

	if vr.apiHandler == nil {
		if vr.Type != HANDLER {
			// 为根节点或分组节点时
			// 为分组类节点添加空操作
			vr.apiHandler = NilApiHandler("?")

		} else {
			// 移除无效的操作类节点
			parent := vr.Parent
			if parent == nil {
				return
			}

			for i, child := range parent.Children {
				if child == vr {
					parent.Children = append(parent.Children[:i], parent.Children[i+1:]...)
					return
				}
			}
			return
		}
	}

	// 设置节点path和params
	vr.setParamsAndPath()
	sort.Sort(vr.Children)
	for _, child := range vr.Children {
		child.Parent = vr
		child.initFromConfig()
	}
}

// 格式化路由
// 设置节点及其子节点的path和params
// 根据path排序同级节点
func (vr *VirtRouter) reset() {
	vr.setParamsAndPathForTree()
	if vr.Parent != nil {
		sort.Sort(vr.Parent.Children)
	}
	vr.sort()
}

// 设置节点及其子节点的path和params
func (vr *VirtRouter) setParamsAndPathForTree() {
	vr.setParamsAndPath()
	for _, child := range vr.Children {
		child.setParamsAndPathForTree()
	}
}

// 设置节点path和params
func (vr *VirtRouter) setParamsAndPath() {
	paramMap := make(map[string]Param)
	pk := []string{}

	// 继承父节点信息
	var parentPath = "/"
	if vr.Parent != nil {
		parentPath = vr.Parent.path
		for _, p := range vr.Parent.params {
			k := p.In + p.Name
			paramMap[k] = p
			pk = append(pk, k)
		}
	}
	// 处理中间件中参数信息
	for _, m := range vr.Middlewares {
		err := m.initApiMiddleware()
		if err != nil {
			Log.Error(err.Error())
			continue
		}
		for _, p := range m.GetApiMiddleware().Params {
			k := p.In + p.Name
			had, ok := paramMap[k]
			if ok {
				p.Required = had.Required || p.Required
			} else {
				pk = append(pk, k)
			}
			paramMap[k] = p
		}
	}
	// 处理操作中参数信息
	if vr.apiHandler != nil {
		for _, p := range vr.apiHandler.Params {
			k := p.In + p.Name
			had, ok := paramMap[k]
			if ok {
				p.Required = had.Required || p.Required
			} else {
				pk = append(pk, k)
			}
			paramMap[k] = p
		}
	}
	// 设置节点最终参数列表
	vr.params = make([]Param, len(pk))
	vr.suffix = ""
	for k, v := range pk {
		vr.params[k] = paramMap[v]
		// 设置URL中path参数
		if vr.params[k].In == "path" {
			vr.params[k].Required = true //path参数不可缺省
			if vr.Type == HANDLER {
				vr.suffix += "/:" + vr.params[k].Name
			}
		}
	}
	// 设置节点path
	if vr.Type == HANDLER {
		vr.path = pathpkg.Join("/", parentPath, vr.Prefix, vr.suffix)
	} else {
		vr.path = pathpkg.Join("/", parentPath, vr.Prefix)
	}
}

func (v *VirtRouter) sort() {
	sort.Sort(v.Children)
	for _, child := range v.Children {
		child.sort()
	}
}

// 注册真实路由
func (vr *VirtRouter) route(g *Group) {
	if !vr.Enable {
		return
	}
	mws := getMiddlewareFuncs(vr.Middlewares)
	prefix := pathpkg.Join("/", vr.Prefix, vr.suffix)
	prefix2 := pathpkg.Join("/", strings.TrimSuffix(vr.Prefix, "/index"), vr.suffix)
	omitIndex := prefix2 != prefix && vr.suffix == ""
	switch vr.Type {
	case GROUP:
		var childGroup *Group
		if omitIndex {
			// "/index"分组会被默认为"/"
			childGroup = g.group(prefix2, mws...)
		} else {
			childGroup = g.group(prefix, mws...)
		}
		for _, child := range vr.Children {
			child.route(childGroup)
		}
	case HANDLER:
		if omitIndex {
			g.match(vr.Methods(), prefix2, vr.apiHandler.Handler, mws...)
		}
		g.match(vr.Methods(), prefix, vr.apiHandler.Handler, mws...)
	}
}

// 虚拟路由配置文件数据结构
type virtRouterConfig struct {
	Md5        string      `json:"md5"`
	VirtRouter *VirtRouter `json:"virtrouter"`
}

// 标记源码路由初始化完成
var canSaveVirtRouterConfig bool

// 读取虚拟路由配置
func readVirtRouterConfig() (md5 string, vr *VirtRouter, err error) {
	f, err := os.OpenFile(ROUTERCONFIG_FILE, os.O_CREATE|os.O_RDONLY, 0777)
	if err != nil {
		return
	}
	defer f.Close()
	b, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}
	if !bytes.Contains(b, utils.String2Bytes("{")) {
		return "", nil, nil
	}
	vrc := virtRouterConfig{}
	err = json.Unmarshal(b, &vrc)
	if err != nil {
		return
	}
	return vrc.Md5, vrc.VirtRouter, err
}

// 保存虚拟路由配置到配置文件
func saveVirtRouterConfig() error {
	if !canSaveVirtRouterConfig {
		// 源码路由初始化未完成时不做保存操作
		return nil
	}
	f, err := os.OpenFile(ROUTERCONFIG_FILE, os.O_CREATE|os.O_WRONLY, 0777)
	if err != nil {
		return err
	}
	defer f.Close()

	b, err := json.MarshalIndent(virtRouterConfig{
		Md5:        Md5,
		VirtRouter: lessgo.virtRouter,
	}, "", "  ")
	if err != nil {
		return err
	}

	f.Write(b)
	return nil
}

type virtRouterSlice []*VirtRouter

func (vs virtRouterSlice) Len() int {
	return len(vs)
}

func (vs virtRouterSlice) Less(i, j int) bool {
	return vs[i].path <= vs[j].path
}

func (vs virtRouterSlice) Swap(i, j int) {
	vs[i], vs[j] = vs[j], vs[i]
}

// 快速返回指定id对于的虚拟路由节点
func GetVirtRouter(id string) (*VirtRouter, bool) {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	vr, ok := virtRouterMap[id]
	return vr, ok
}

// 创建虚拟路由动态分组
func NewGroupVirtRouter(prefix, desc string) *VirtRouter {
	prefix = cleanPrefix(prefix)
	ah := NilApiHandler(desc)
	return &VirtRouter{
		Id:          uuid.New().String(),
		Type:        GROUP,
		Prefix:      prefix,
		Enable:      true,
		Dynamic:     true,
		apiHandler:  ah,
		Middlewares: []*MiddlewareConfig{},
		Hid:         ah.id,
	}
}

// 创建虚拟路由动态操作
func NewHandlerVirtRouter(prefix, hid string, middlewares ...*MiddlewareConfig) (*VirtRouter, error) {
	prefix = strings.Split(prefix, "/:")[0]
	vr := &VirtRouter{
		Id:          uuid.New().String(),
		Type:        HANDLER,
		Prefix:      prefix,
		Enable:      true,
		Dynamic:     true,
		Middlewares: middlewares,
		Hid:         hid,
	}
	err := vr.SetApiHandler(hid)
	return vr, err
}

// 创建虚拟路由根节点
func newRootVirtRouter() *VirtRouter {
	ah := NilApiHandler("root")
	root := &VirtRouter{
		Id:          uuid.New().String(),
		Type:        ROOT,
		Prefix:      "/",
		Dynamic:     false,
		Enable:      true,
		apiHandler:  ah,
		Middlewares: []*MiddlewareConfig{},
		Hid:         ah.id,
	}
	root.reset()
	return root
}

// 从配置文件初始化虚拟路由
func initVirtRouterConfig() {
	md5, vr, err := readVirtRouterConfig()
	if err != nil {
		Log.Error("Read the config/virtrouter.config failed: %v.", err)
		return
	}

	// 重新运行程序
	if md5 == Md5 {
		if vr != nil {
			vr.initFromConfig()
			lessgo.virtRouter = vr
		}
		canSaveVirtRouterConfig = true
		return
	}

	defer func() {
		// 标记源码路由初始化完成
		canSaveVirtRouterConfig = true
		// 覆盖保存配置
		err := saveVirtRouterConfig()
		if err != nil {
			Log.Error("Save the config/virtrouter.config failed: %v.", err)
		}
	}()

	// 第一次运行程序
	if md5 == "" {
		return
	}

	// 程序被重新编译后第一次运行
	if vr != nil {
		vr.initFromConfig()
		merge(lessgo.virtRouter, vr)
		os.Remove(ROUTERCONFIG_FILE)
	}
	return
}

// 虚拟路由树同级节点合并，将b合并入a，冲突时以a为准
func merge(a, b *VirtRouter) {
	// 类型不同不合并
	if a.Type != b.Type {
		return
	}

	// 类型为操作时
	if a.Type == HANDLER {
		if a.Prefix != b.Prefix || a.apiHandler.Method != b.apiHandler.Method {
			b.Dynamic = true     //强制为动态的配置路由
			a.Parent.addChild(b) //并入正式虚拟路由树中
		}
		return
	}

	// 类型为根路由或分组时

	// 路由前缀不同时可合并
	if a.Prefix != b.Prefix {
		b.Dynamic = true     //强制为动态的配置路由
		a.Parent.addChild(b) //并入正式虚拟路由树中
		return
	}

	var has *VirtRouter
	for _, bc := range b.Children {
		has = nil
		for _, ac := range a.Children {
			if ac.Prefix == b.Prefix {
				has = ac
				break
			}
		}
		if has == nil {
			// 当前源码路由中不存在的节点，直接增加
			b.Dynamic = true     //强制为动态的配置路由
			a.Parent.addChild(b) //并入正式虚拟路由树中
		} else {
			merge(bc, has)
		}
	}
}

// 添加路由节点
func addVirtRouter(vr *VirtRouter) bool {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	if _, ok := virtRouterMap[vr.Id]; ok {
		return false
	}
	virtRouterMap[vr.Id] = vr
	return true
}

// 删除路由节点
func delVirtRouter(vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, vr.Id)
}

// 格式化path前缀
func cleanPrefix(prefix string) string {
	prefix = strings.Split(prefix, ":")[0]
	return pathpkg.Join("/", prefix)
}
