package lessgo

import (
	"fmt"
	pathpkg "path"
	"strings"
	"sync"

	"github.com/go-xorm/xorm"

	"github.com/lessgo/lessgo/utils/uuid"
)

// 虚拟路由
type VirtRouter struct {
	Id          string             `json:"id" xorm:"not null pk VARCHAR(36)"`   // UUID
	Pid         string             `json:"pid" xorm:"VARCHAR(36)"`              // 父节点id
	Type        int                `json:"type" xorm:"not null TINYINT(1)"`     // 操作类型: 根目录/路由分组/操作
	Prefix      string             `json:"prefix" xorm:"not null VARCHAR(500)"` // 路由节点的url前缀(不含参数)
	Middlewares []MiddlewareConfig `json:"middlewares" xorm:"TEXT json"`        // 中间件列表 (允许运行时修改)
	Enable      bool               `json:"enable" xorm:"not null TINYINT(1)"`   // 是否启用当前路由节点
	Dynamic     bool               `json:"dynamic" xorm:"not null TINYINT(1)"`  // 是否动态追加的节点
	Hid         string             `json:"hid" xorm:"not null VARCHAR(500)"`    // 操作ApiHandler.id

	path       string        `xorm:"-"` // 路由匹配模式
	prefixPath string        `xorm:"-"` // 路由节点的url前缀的固定路径部分
	parent     *VirtRouter   `xorm:"-"` // 父节点
	children   []*VirtRouter `xorm:"-"` // 子节点
	apiHandler *ApiHandler   `xorm:"-"` // 操作
}

type VirtRouterLock struct {
	Md5 string `json:"Md5" xorm:"not null VARCHAR(500)"`
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

	// 非动态节点不可修改
	notDynamicError = fmt.Errorf("The specified node is not dynamic, and therefore can not be modified.")
)

// 从数据库初始化虚拟路由
func initVirtRouterFromDB() (err error) {
	lessgodb, _ = GetDB("lessgo")
	if lessgodb == nil {
		Logger().Warn("The lessgo database can not be used, now only use source code routing.\n")
		return
	}
	if err = lessgodb.Ping(); err != nil {
		Logger().Error("%v", err)
		Logger().Warn("The lessgo database connection failed, now only use source code routing.\n")
		return nil
	}
	vrlock := new(VirtRouterLock)
	err = lessgodb.Sync2(vrlock)
	if err != nil {
		err = fmt.Errorf("Failed to sync table virt_router_lock: %v.", err)
		return
	}
	vr := new(VirtRouter)
	has, err := lessgodb.Get(vrlock)
	if vrlock.Md5 != Md5 {
		exist, err := lessgodb.IsTableExist(vr)
		if exist || err != nil {
			err = lessgodb.DropTables(vr)
			if err != nil {
				err = fmt.Errorf("Failed to drop table virt_router: %v.", err)
				return err
			}
		}
	}
	err = lessgodb.Sync2(vr)
	if err != nil {
		err = fmt.Errorf("Failed to sync table virt_router: %v.", err)
		return
	}
	session := lessgodb.NewSession()
	defer session.Close()
	err = session.Begin()
	if err != nil {
		return
	}

	if !has {

		// 首次运行当前软件时
		vrlock.Md5 = Md5
		_, err = session.Insert(vrlock)
		if err != nil {
			session.Rollback()
			vrlock.Md5 = ""
			err = fmt.Errorf("Failed to insert table virt_router md5: %v.", err)
			return
		}

		err = dbReset(session)
		if err != nil {
			return
		}

	} else if vrlock.Md5 != Md5 {

		// 软件重新编译后再次运行时
		vrlock.Md5 = Md5
		_, err = session.Update(vrlock)
		if err != nil {
			session.Rollback()
			vrlock.Md5 = ""
			err = fmt.Errorf("Failed to update table virt_router md5: %v.", err)
			return
		}

		var dbInfo []*VirtRouter
		err = session.Find(&dbInfo)
		if err != nil {
			session.Rollback()
			err = fmt.Errorf("Failed to read table virt_router: %v.", err)
			return
		}
		if len(dbInfo) > 0 {
			// 构建历史版本的虚拟路由树
			dbRootVirtRouter := buildVirtRouter(dbInfo)
			merge(DefLessgo.virtRouter, dbRootVirtRouter)
			virtRouterLock.Lock()
			defer virtRouterLock.Unlock()
			virtRouterMap = map[string]*VirtRouter{}
			vrs := DefLessgo.virtRouter.Progeny()
			for _, vr := range vrs {
				virtRouterMap[vr.Id] = vr
			}
		}
		err = dbReset(session)
		if err != nil {
			return
		}

	} else {

		// 软件重复运行时
		var dbInfo []*VirtRouter
		err = session.Find(&dbInfo)
		if err != nil {
			session.Rollback()
			err = fmt.Errorf("Failed to read table virt_router: %v.", err)
			return
		}
		if len(dbInfo) == 0 {
			session.Rollback()
			err = fmt.Errorf("Table virt_router does not exist.")
			return
		}
		// 从配置信息构建虚拟路由树
		DefLessgo.virtRouter = buildVirtRouter(dbInfo)
	}
	err = session.Commit()
	return
}

// 重建数据库中虚拟路由配置信息
func dbReset(session *xorm.Session) (err error) {
	nodes := DefLessgo.virtRouter.Progeny()
	_, err = session.Insert(&nodes)
	if err != nil {
		session.Rollback()
		err = fmt.Errorf("Failed to insert table virt_router: %v.", err)
	}
	return
}

// 虚拟路由树同级节点合并，将b合并入a，冲突时以a为准
func merge(a, b *VirtRouter) {
	if a.Prefix != b.Prefix {
		if a.parent != nil {
			b.Dynamic = true     //强制为动态的配置路由
			a.parent.addChild(b) //并入正式虚拟路由树中
		}
		return
	} else {
		for _, ac := range a.children {
			for _, bc := range b.children {
				merge(ac, bc)
			}
		}
	}
}

// 构建虚拟路由树
func buildVirtRouter(vrs []*VirtRouter) *VirtRouter {
	virtRouterMap2 := map[string]*VirtRouter{}
	for _, vr := range vrs {
		vr.apiHandler = getApiHandler(vr.Hid)
		virtRouterMap2[vr.Id] = vr
	}
	var root *VirtRouter
	for _, vr := range virtRouterMap2 {
		if vr.Type == ROOT {
			root = vr
		}
		parent := virtRouterMap2[vr.Pid]
		if parent == nil {
			continue
		}
		vr.parent = parent
		parent.children = append(parent.children, vr)
	}
	root.reset()
	return root
}

// 创建虚拟路由根节点
func newRootVirtRouter() *VirtRouter {
	ah := NilApiHandler("Service Root")
	root := &VirtRouter{
		Id:          uuid.New().String(),
		Type:        ROOT,
		Prefix:      "/",
		Dynamic:     false,
		Enable:      true,
		apiHandler:  ah,
		Middlewares: []MiddlewareConfig{},
		Hid:         ah.id,
	}
	root.reset()
	return root
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
		Middlewares: []MiddlewareConfig{},
		Hid:         ah.id,
	}
}

// 创建虚拟路由动态操作
func NewHandlerVirtRouter(prefix, hid string, middlewares ...MiddlewareConfig) (*VirtRouter, error) {
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

// 快速返回指定id对于的虚拟路由节点
func GetVirtRouter(id string) (*VirtRouter, bool) {
	virtRouterLock.RLock()
	defer virtRouterLock.RUnlock()
	vr, ok := virtRouterMap[id]
	return vr, ok
}

// 返回操作中定义的方法字符串("WS"和"*"不做转换)
func (vr *VirtRouter) Method() string {
	return vr.apiHandler.Method
}

// 真实的请求方法列表(自动转换: "WS"->"GET", "*"->methods)
func (vr *VirtRouter) Methods() []string {
	return vr.apiHandler.Methods()
}

// 操作的描述
func (vr *VirtRouter) Description() string {
	return vr.apiHandler.Desc
}

// 操作的参数说明列表的副本
func (vr *VirtRouter) Params() []Param {
	return vr.apiHandler.Params
}

// // 操作接受的响应内容类型
// func (vr *VirtRouter) Produces() []string {
// 	return vr.apiHandler.Produces
// }

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

// 设置虚拟路由节点url前缀
func (vr *VirtRouter) SetPrefix(prefix string) (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	if lessgodb == nil {
		goto label
	}
	_, err = lessgodb.Where("id=?", vr.Id).Cols("prefix").Update(&VirtRouter{Prefix: prefix})
	if err != nil {
		return
	}
label:
	vr.Prefix = prefix
	return
}

func (vr *VirtRouter) Suffix() string {
	return vr.apiHandler.Suffix()
}

// 启用/禁用虚拟路由节点
func (vr *VirtRouter) SetEnable(able bool) (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	if lessgodb == nil {
		goto label
	}
	_, err = lessgodb.Where("id=?", vr.Id).Cols("enable").Update(&VirtRouter{Enable: able})
	if err != nil {
		return
	}
label:
	vr.Enable = able
	return
}

// 配置中间件（仅在源码中使用）
func (vr *VirtRouter) Use(middlewares ...*ApiMiddleware) *VirtRouter {
	if vr.Dynamic {
		Logger().Error("Specified node is dynamic, please use ResetUse(middlewares []string) (err error).")
		return vr
	}
	l := len(middlewares)
	if l == 0 {
		return vr
	}
	_l := len(vr.Middlewares)
	ms := make([]MiddlewareConfig, _l+l)
	copy(ms, vr.Middlewares)
	for i, m := range middlewares {
		m.init()
		ms[i+_l] = MiddlewareConfig{
			Name:   m.Name,
			Config: m.defaultConfig,
		}
	}
	vr.Middlewares = ms
	return vr
}

// 重置中间件
func (vr *VirtRouter) ResetUse(middlewares []MiddlewareConfig) (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	if middlewares == nil {
		middlewares = []MiddlewareConfig{}
	}
	if lessgodb == nil {
		goto label
	}
	_, err = lessgodb.Where("id=?", vr.Id).Cols("middlewares").Update(&VirtRouter{Middlewares: middlewares})
	if err != nil {
		return
	}
label:
	vr.Middlewares = middlewares
	return
}

// 为节点更换操作
func (vr *VirtRouter) SetApiHandler(hid string) error {
	if !vr.Dynamic {
		return notDynamicError
	}
	vh := getApiHandler(hid)
	if vh == nil {
		return fmt.Errorf("Specified ApiHandler does not exist.")
	}
	vr.Hid = hid
	vr.apiHandler = vh
	vr.reset()
	return nil
}

// 所有子节点的列表副本
func (vr *VirtRouter) Children() []*VirtRouter {
	cr := make([]*VirtRouter, len(vr.children))
	copy(cr, vr.children)
	return cr
}

// 添加子节点
func (vr *VirtRouter) AddChild(virtRouter *VirtRouter) (err error) {
	if !virtRouter.Dynamic {
		return notDynamicError
	}
	return vr.addChild(virtRouter)
}

func (vr *VirtRouter) addChild(virtRouter *VirtRouter) (err error) {
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
	if !virtRouter.Dynamic {
		return notDynamicError
	}
	return vr.delChild(virtRouter)
}

func (vr *VirtRouter) delChild(virtRouter *VirtRouter) (err error) {
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
	return fmt.Errorf("node %v does not have child node: %v.", vr.Description(), virtRouter.Description())
}

// 虚拟路由节点的父节点
func (vr *VirtRouter) Parent() *VirtRouter {
	return vr.parent
}

// 删除自身
func (vr *VirtRouter) Delete() (err error) {
	if !vr.Dynamic {
		return notDynamicError
	}
	return vr.delete()
}

func (vr *VirtRouter) delete() error {
	if vr.parent != nil {
		vr.parent.delChild(vr)
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
	var suffix string
	if vr.apiHandler != nil {
		suffix = vr.apiHandler.Suffix()
	}
	vr.path = pathpkg.Join("/", parentPath, vr.Prefix, suffix)
	for _, child := range vr.children {
		child.reset()
	}
}

/*
 * 注册真实路由
 */
func (vr *VirtRouter) route(group *Group) {
	if !vr.Enable {
		return
	}
	mws := getMiddlewareFuncs(vr.Middlewares)
	prefix := pathpkg.Join("/", vr.Prefix, vr.apiHandler.Suffix())
	prefix2 := pathpkg.Join("/", strings.TrimSuffix(vr.Prefix, "/index"), vr.apiHandler.Suffix())
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
		var ms []string
		if vr.Method() == WS {
			ms = []string{WS}
		} else {
			ms = vr.Methods()
		}
		if hasIndex {
			group.Match(ms, prefix2, vr.apiHandler.Handler, mws...)
		}
		group.Match(ms, prefix, vr.apiHandler.Handler, mws...)
	}
}

func registerVirtRouter() {
	if err := isExistMiddlewares(DefLessgo.before...); err != nil {
		Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err := isExistMiddlewares(DefLessgo.after...); err != nil {
		Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err := isExistMiddlewares(DefLessgo.prefix...); err != nil {
		Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}
	if err := isExistMiddlewares(DefLessgo.suffix...); err != nil {
		Logger().Error("Create/Recreate the router is faulty: %v", err)
		return
	}

	// 从虚拟路由创建真实路由
	DefLessgo.app.router = NewRouter(DefLessgo.app)
	DefLessgo.app.middleware = []MiddlewareFunc{DefLessgo.app.router.Process}
	DefLessgo.app.head = DefLessgo.app.pristineHead
	DefLessgo.app.BeforeUse(getMiddlewareFuncs(DefLessgo.before)...)
	DefLessgo.app.AfterUse(getMiddlewareFuncs(DefLessgo.after)...)
	DefLessgo.app.PreUse(getMiddlewareFuncs(DefLessgo.prefix)...)
	DefLessgo.app.SufUse(getMiddlewareFuncs(DefLessgo.suffix)...)
	group := DefLessgo.app.Group(
		DefLessgo.virtRouter.Prefix,
		getMiddlewareFuncs(DefLessgo.virtRouter.Middlewares)...,
	)
	for _, child := range DefLessgo.virtRouter.Children() {
		child.route(group)
	}
}

func resetVirtRouter(oldId string, vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, oldId)
	virtRouterMap[vr.Id] = vr
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

func delVirtRouter(vr *VirtRouter) {
	virtRouterLock.Lock()
	defer virtRouterLock.Unlock()
	delete(virtRouterMap, vr.Id)
}

func cleanPrefix(prefix string) string {
	prefix = strings.Split(prefix, ":")[0]
	return pathpkg.Join("/", prefix)
}
