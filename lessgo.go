/*
Package lessgo implements a simple, stable, efficient and flexible web framework for Go.

Author1: https://github.com/henrylee2cn
Author2: https://github.com/changyu72
*/
package lessgo

import (
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/go-xorm/xorm"

	_ "github.com/lessgo/lessgo/_fixture"
	"github.com/lessgo/lessgo/dbservice"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/utils"
	"github.com/lessgo/lessgo/utils/uuid"
)

type (
	lessgo struct {
		app       *Echo
		AppConfig *Config
		dbService *dbservice.DBService

		//全局操作列表
		apiHandlers []*ApiHandler

		//全局中间件列表
		apiMiddlewares []*ApiMiddleware

		// 路由执行前后的中间件登记
		before []MiddlewareConfig //路由执行前中间件
		after  []MiddlewareConfig //路由执行后中间件
		prefix []MiddlewareConfig //第一批执行的中间件
		suffix []MiddlewareConfig //最后一批执行的中间件

		// 用于构建最终真实路由的虚拟路由；
		// 初始值为源码中定义的路由，之后追加配置中定义的路由；
		// 配置路由为空时，复制源码中定义的路由到配置路由；
		// 再次运行时，直接读取配置路由覆盖自身；
		// 源码因修改产生冲突时，以源码路由为主；
		// 修改配置路由时，不允许和源码路由冲突；
		// 源码路由只允许在配置中增加子节点。
		virtRouter *VirtRouter

		home         string //根路径"/"对应的url
		serverEnable bool   //服务是否启用
		lock         sync.RWMutex
	}
)

const (
	NAME    = "Lessgo"
	VERSION = "0.6.0"
	ADDRESS = "https://github.com/lessgo/lessgo"
)

const (
	MB = 1 << 20
)

// 初始化全局Lessgo实例
var DefLessgo = newLessgo()

/*
 * 获取lessgo全局实例
 */
func Lessgo() *lessgo {
	return DefLessgo
}

/*
 * 设置主页
 */
func SetHome(homeurl string) {
	if DefLessgo.AppConfig.RouterCaseSensitive {
		DefLessgo.home = homeurl
	} else {
		DefLessgo.home = strings.ToLower(homeurl)
	}
}

/*
 * 返回设置的主页
 */
func GetHome() string {
	return DefLessgo.home
}

/*
 * 开启网站服务
 */
func EnableServer() {
	DefLessgo.lock.Lock()
	DefLessgo.serverEnable = true
	DefLessgo.lock.Unlock()
}

/*
 * 关闭网站服务
 */
func DisableServer() {
	DefLessgo.lock.Lock()
	DefLessgo.serverEnable = false
	DefLessgo.lock.Unlock()
}

/*
 * 查询网站服务状态
 */
func ServerEnable() bool {
	DefLessgo.lock.RLock()
	defer DefLessgo.lock.RUnlock()
	return DefLessgo.serverEnable
}

/*
 * 运行服务
 */
func Run() {
	// 从数据库初始化虚拟路由
	initVirtRouterFromDB()
	// 重建路由
	ReregisterRouter()

	// 开启最大核心数运行
	runtime.GOMAXPROCS(runtime.NumCPU())

	// 配置服务器引擎
	var (
		tlsCertfile string
		tlsKeyfile  string
		mode        string
		graceful    string
		h           = "HTTP"
	)
	if AppConfig.Listen.EnableHTTPS {
		h = "HTTPS"
		tlsCertfile = AppConfig.Listen.HTTPSCertFile
		tlsKeyfile = AppConfig.Listen.HTTPSKeyFile
	}
	if AppConfig.Debug {
		mode = "debug"
	} else {
		mode = "release"
	}
	if AppConfig.Listen.Graceful {
		graceful = "(enable-graceful-restart)"
	} else {
		graceful = "(disable-graceful-restart)"
	}

	Logger().Sys("> %s listening and serving %s on %v (%s-mode) %v", AppConfig.AppName, h, AppConfig.Listen.Address, mode, graceful)

	// 启动服务
	DefLessgo.app.Run(
		AppConfig.Listen.Address,
		tlsCertfile,
		tlsKeyfile,
		time.Duration(AppConfig.Listen.ReadTimeout),
		time.Duration(AppConfig.Listen.WriteTimeout),
		AppConfig.Listen.Graceful,
	)
}

/*
 * 数据库引擎
 */

// 获取默认数据库引擎
func DefaultDB() *xorm.Engine {
	return DefLessgo.dbService.DefaultDB()
}

// 获取指定数据库引擎
func GetDB(name string) (*xorm.Engine, bool) {
	return DefLessgo.dbService.GetDB(name)
}

/*
 * 打印实例
 */
func Logger() logs.Logger {
	return DefLessgo.app.Logger()
}

/*
 * Session管理平台实例
 */
func Sessions() *session.Manager {
	return DefLessgo.app.sessions
}

/*
 * 操作
 */

// 获取已注册的操作列表
func Handlers() []*ApiHandler {
	return DefLessgo.apiHandlers
}

/*
 * 中间件
 */

// 获取已注册的中间件列表
func Middlewares() []*ApiMiddleware {
	return DefLessgo.apiMiddlewares
}

/*
 * 虚拟路由
 */

// 虚拟路由列表
func RouterList() []*VirtRouter {
	return DefLessgo.virtRouter.Progeny()
}

// 虚拟路由根节点
func RootRouter() *VirtRouter {
	return DefLessgo.virtRouter
}

// 操作列表（禁止修改）
func ApiHandlerList() []*ApiHandler {
	return DefLessgo.apiHandlers
}

// 在路由执行位置之前紧邻插入中间件队列
func BeforeUse(middleware ...MiddlewareConfig) {
	DefLessgo.before = append(DefLessgo.before, middleware...)
}

// 在路由执行位置之后紧邻插入中间件队列
func AfterUse(middleware ...MiddlewareConfig) {
	DefLessgo.after = append(middleware, DefLessgo.after...)
}

// 第一批执行的中间件
func PreUse(middleware ...MiddlewareConfig) {
	DefLessgo.prefix = append(DefLessgo.prefix, middleware...)
}

// 最后一批执行的中间件
func SufUse(middleware ...MiddlewareConfig) {
	DefLessgo.suffix = append(middleware, DefLessgo.suffix...)
}

// 从根路由开始配置路由(必须在init()中调用)
func Root(nodes ...*VirtRouter) {
	var err error
	for _, node := range nodes {
		if node == nil {
			continue
		}
		err = DefLessgo.virtRouter.addChild(node)
		if err != nil {
			Logger().Error("%v", err)
		}
	}
}

// 配置路由分组(必须在init()中调用)
func Branch(prefix, desc string, nodes ...*VirtRouter) *VirtRouter {
	parent := NewGroupVirtRouter(prefix, desc)
	parent.Dynamic = false
	var err error
	for _, node := range nodes {
		if node == nil {
			continue
		}
		err = parent.addChild(node)
		if err != nil {
			Logger().Error("%v", err)
		}
	}
	return parent
}

// 配置路由操作(必须在init()中调用)
func Leaf(prefix string, apiHandler *ApiHandler, middlewares ...*ApiMiddleware) *VirtRouter {
	prefix = cleanPrefix(prefix)
	ms := make([]MiddlewareConfig, len(middlewares))
	for i, m := range middlewares {
		m.init()
		ms[i] = MiddlewareConfig{
			Name:   m.Name,
			Config: m.defaultConfig,
		}
	}
	vr := &VirtRouter{
		Id:          uuid.New().String(),
		Type:        HANDLER,
		Prefix:      prefix,
		Enable:      true,
		Dynamic:     false,
		Middlewares: ms,
		apiHandler:  apiHandler.init(),
		Hid:         apiHandler.id,
	}
	return vr
}

// 创建静态目录服务的操作
func StaticFunc(root string) HandlerFunc {
	return func(c Context) error {
		return c.File(path.Join(root, c.P(0)))
	}
}

/*
 * 重建真实路由
 */
func ReregisterRouter() {
	DefLessgo.app.lock.Lock()
	defer DefLessgo.app.lock.Unlock()
	registerVirtRouter()
	registerStaticRouter()
}

/*
 * 软件自身md5
 */
var Md5 = func() string {
	file, _ := exec.LookPath(os.Args[0])
	info, _ := os.Stat(file)
	return utils.MakeUnique(info.ModTime())
}()
