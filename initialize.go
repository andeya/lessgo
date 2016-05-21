package lessgo

import (
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"time"

	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"

	"github.com/lessgo/lessgo/dbservice"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/utils"
)

func newLessgo() *lessgo {
	printInfo()
	err := LoadAppConfig()
	if err != nil {
		fmt.Println(err)
	}
	err = LoadDBConfig()
	if err != nil {
		fmt.Println(err)
	}
	registerMime()

	l := &lessgo{
		app:            New(),
		AppConfig:      AppConfig,
		home:           "/",
		serverEnable:   true,
		apiHandlers:    []*ApiHandler{},
		apiMiddlewares: []*ApiMiddleware{},
		before:         []*MiddlewareConfig{},
		after:          []*MiddlewareConfig{},
		virtRouter:     newRootVirtRouter(),
	}

	// 初始化日志
	l.app.Logger().SetMsgChan(AppConfig.Log.AsyncChan)
	l.app.SetLogLevel(AppConfig.Log.Level)

	// 设置运行模式
	l.app.SetDebug(AppConfig.Debug)

	// 设置静态资源缓存
	l.app.SetMemoryCache(NewMemoryCache(
		AppConfig.FileCache.SingleFileAllowMB*MB,
		AppConfig.FileCache.MaxCapMB*MB,
		time.Duration(AppConfig.FileCache.CacheSecond)*time.Second),
	)

	// 设置渲染接口
	l.app.SetRenderer(NewPongo2Render(AppConfig.Debug))

	// 设置上传文件允许的最大尺寸
	MaxMemory = AppConfig.MaxMemoryMB * MB

	// 配置数据库
	l.dbService = registerDBService()

	// 初始化全局session
	err = registerSession()
	if err != nil {
		l.app.Logger().Error("Failed to create GlobalSessions: %v.", err)
	}
	if GlobalSessions == nil {
		l.app.Logger().Sys("Session is disable.")
	} else {
		l.app.SetSessions(GlobalSessions)
		l.app.Logger().Sys("Session is enable.")
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

func registerSession() (err error) {
	if !AppConfig.Session.SessionOn {
		GlobalSessions = nil
		return
	}
	conf := map[string]interface{}{
		"cookieName":              AppConfig.Session.SessionName,
		"gclifetime":              AppConfig.Session.SessionGCMaxLifetime,
		"providerConfig":          filepath.ToSlash(AppConfig.Session.SessionProviderConfig),
		"secure":                  AppConfig.Listen.EnableHTTPS,
		"enableSetCookie":         AppConfig.Session.SessionAutoSetCookie,
		"domain":                  AppConfig.Session.SessionDomain,
		"cookieLifeTime":          AppConfig.Session.SessionCookieLifeTime,
		"enableSidInHttpHeader":   AppConfig.Session.EnableSidInHttpHeader,
		"sessionNameInHttpHeader": AppConfig.Session.SessionNameInHttpHeader,
		"enableSidInUrlQuery":     AppConfig.Session.EnableSidInUrlQuery,
	}
	confBytes, _ := json.Marshal(conf)
	GlobalSessions, err = session.NewManager(AppConfig.Session.SessionProvider, string(confBytes))
	if err != nil {
		return
	}
	go GlobalSessions.GC()
	return
}

// 注册固定的静态文件与目录
func registerStaticRouter() {
	DefLessgo.app.Static("/uploads", UPLOADS_DIR, autoHTMLSuffix)
	DefLessgo.app.Static("/static", STATIC_DIR, filterTemplate(), autoHTMLSuffix)
	DefLessgo.app.Static("/biz", BIZ_VIEW_DIR, filterTemplate(), autoHTMLSuffix)
	DefLessgo.app.Static("/sys", SYS_VIEW_DIR, filterTemplate(), autoHTMLSuffix)

	DefLessgo.app.File("/favicon.ico", IMG_DIR+"/favicon.ico")
}

// 设置系统预设的中间件
func registerMiddleware() {
	PreUse(
		&MiddlewareConfig{Name: "检查服务器是否启用"},
		&MiddlewareConfig{Name: "检查是否为访问主页"},
		&MiddlewareConfig{Name: "系统运行日志打印"},
		&MiddlewareConfig{Name: "捕获运行时恐慌"},
	)
	if AppConfig.CrossDomain {
		BeforeUse(&MiddlewareConfig{Name: "设置允许跨域"})
	}
}

// 注册数据库服务
func registerDBService() *dbservice.DBService {
	dbService := &dbservice.DBService{
		List: map[string]*xorm.Engine{},
	}
	for _, conf := range AppConfig.DBList {
		engine, err := xorm.NewEngine(conf.Driver, conf.ConnString)
		if err != nil {
			logs.Error("%v\n", err)
			continue
		}
		logger := dbservice.NewILogger(AppConfig.Log.AsyncChan, AppConfig.Log.Level, conf.Name)
		logger.BeeLogger.EnableFuncCallDepth(AppConfig.Debug)

		engine.SetLogger(logger)
		engine.SetMaxOpenConns(conf.MaxOpenConns)
		engine.SetMaxIdleConns(conf.MaxIdleConns)
		engine.SetDisableGlobalCache(conf.DisableCache)
		engine.ShowSQL(conf.ShowSql)
		engine.ShowExecTime(conf.ShowExecTime)
		if (conf.TableFix == "prefix" || conf.TableFix == "suffix") && len(conf.TableSpace) > 0 {
			var impr core.IMapper
			if conf.TableSnake {
				impr = core.SnakeMapper{}
			} else {
				impr = core.SameMapper{}
			}
			if conf.TableFix == "prefix" {
				engine.SetTableMapper(core.NewPrefixMapper(impr, conf.TableSpace))
			} else {
				engine.SetTableMapper(core.NewSuffixMapper(impr, conf.TableSpace))
			}
		}
		if (conf.ColumnFix == "prefix" || conf.ColumnFix == "suffix") && len(conf.ColumnSpace) > 0 {
			var impr core.IMapper
			if conf.ColumnSnake {
				impr = core.SnakeMapper{}
			} else {
				impr = core.SameMapper{}
			}
			if conf.ColumnFix == "prefix" {
				engine.SetTableMapper(core.NewPrefixMapper(impr, conf.ColumnSpace))
			} else {
				engine.SetTableMapper(core.NewSuffixMapper(impr, conf.ColumnSpace))
			}
		}

		if conf.Driver == "sqlite3" && !utils.FileExists(conf.ConnString) {
			os.MkdirAll(filepath.Dir(conf.ConnString), 0777)
			f, err := os.Create(conf.ConnString)
			if err != nil {
				logs.Global.Error("%v", err)
			} else {
				f.Close()
			}
		}

		dbService.List[conf.Name] = engine
		if AppConfig.DefaultDB == conf.Name {
			dbService.Default = engine
		}
	}
	return dbService
}
