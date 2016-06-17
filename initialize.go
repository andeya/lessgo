package lessgo

import (
	"encoding/json"
	"fmt"
	"mime"
	"path/filepath"
	"time"

	"github.com/lessgo/lessgo/session"
)

func newLessgo() *Lessgo {
	printInfo()

	err := Config.LoadMainConfig()
	if err != nil {
		fmt.Println(err)
	}

	registerMime()

	l := &Lessgo{
		App:            app,
		config:         Config,
		home:           "/",
		serverEnable:   true,
		apiHandlers:    []*ApiHandler{},
		apiMiddlewares: []*ApiMiddleware{},
		virtBefore:     []*MiddlewareConfig{},
		virtAfter:      []*MiddlewareConfig{},
		virtStatics:    []*VirtStatic{},
		virtFiles:      []*VirtFile{},
		virtRouter:     newRootVirtRouter(),
	}

	// 初始化全局日志
	Log.SetMsgChan(Config.Log.AsyncChan)
	Log.SetLevel(Config.Log.Level)

	// 设置运行模式
	l.App.SetDebug(Config.Debug)

	// 设置静态资源缓存
	l.App.setMemoryCache(NewMemoryCache(
		Config.FileCache.SingleFileAllowMB*MB,
		Config.FileCache.MaxCapMB*MB,
		time.Duration(Config.FileCache.CacheSecond)*time.Second),
	)

	// 设置渲染接口
	l.App.SetRenderer(NewPongo2Render(!Config.Debug))

	// 设置上传文件允许的最大尺寸
	MaxMemory = Config.MaxMemoryMB * MB

	// 初始化sessions管理实例
	sessions, err := newSessions()
	if err != nil {
		Log.Error("Failed to create sessions: %v.", err)
	}
	if sessions == nil {
		Log.Sys("Session is disable.")
	} else {
		go sessions.GC()
		l.App.setSessions(sessions)
		Log.Sys("Session is enable.")
	}

	return l
}

func printInfo() {
	fmt.Printf(">%s %s (%s)\n", NAME, VERSION, ADDRESS)
}

func registerMime() {
	for k, v := range mimemaps {
		mime.AddExtensionType(k, v)
	}
}

func newSessions() (sessions *session.Manager, err error) {
	if !Config.Session.SessionOn {
		return
	}
	conf := map[string]interface{}{
		"cookieName":              Config.Session.SessionName,
		"gclifetime":              Config.Session.SessionGCMaxLifetime,
		"providerConfig":          filepath.ToSlash(Config.Session.SessionProviderConfig),
		"secure":                  Config.Listen.EnableHTTPS,
		"enableSetCookie":         Config.Session.SessionAutoSetCookie,
		"domain":                  Config.Session.SessionDomain,
		"cookieLifeTime":          Config.Session.SessionCookieLifeTime,
		"enableSidInHttpHeader":   Config.Session.EnableSidInHttpHeader,
		"sessionNameInHttpHeader": Config.Session.SessionNameInHttpHeader,
		"enableSidInUrlQuery":     Config.Session.EnableSidInUrlQuery,
	}
	confBytes, _ := json.Marshal(conf)
	return session.NewManager(Config.Session.SessionProvider, string(confBytes))
}

// 尝试设置系统默认通用操作
func tryRegisterDefaultHandler() {
	if lessgo.App.router.NotFound == nil {
		SetNotFound(defaultNotFoundHandler)
	}
	if lessgo.App.router.MethodNotAllowed == nil {
		SetMethodNotAllowed(defaultMethodNotAllowedHandler)
	}
	if lessgo.App.router.ErrorPanicHandler == nil {
		SetInternalServerError(defaultInternalServerErrorHandler)
	}
}

// 添加系统预设的路由操作前的中间件
func registerBefore() {
	PreUse(
		&MiddlewareConfig{Name: "检查服务器是否启用"},
		&MiddlewareConfig{Name: "检查是否为访问主页"},
		&MiddlewareConfig{Name: "系统运行日志打印"},
		&MiddlewareConfig{Name: "捕获运行时恐慌"},
	)
	if Config.CrossDomain {
		BeforeUse(&MiddlewareConfig{Name: "设置允许跨域"})
	}
}

// 添加系统预设的路由操作后的中间件
func registerAfter() {

}

// 添加系统预设的静态目录虚拟路由
func registerStatics() {
	Static("/uploads", UPLOADS_DIR, AutoHTMLSuffix)
	Static("/static", STATIC_DIR, FilterTemplate, AutoHTMLSuffix)
	Static("/biz", BIZ_VIEW_DIR, FilterTemplate, AutoHTMLSuffix)
	Static("/sys", SYS_VIEW_DIR, FilterTemplate, AutoHTMLSuffix)
}

// 添加系统预设的静态文件虚拟路由
func registerFiles() {
	File("/favicon.ico", IMG_DIR+"/favicon.ico")
}
