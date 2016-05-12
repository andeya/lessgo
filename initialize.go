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

	"github.com/lessgo/lessgo/config"
	"github.com/lessgo/lessgo/dbservice"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/utils"
)

func newLessgo() *lessgo {
	printInfo()
	registerAppConfig()
	registerDBConfig()
	registerMime()

	l := &lessgo{
		app:            New(),
		AppConfig:      AppConfig,
		home:           "/",
		serverEnable:   true,
		apiHandlers:    []*ApiHandler{},
		apiMiddlewares: []*ApiMiddleware{},
		before:         []MiddlewareConfig{},
		after:          []MiddlewareConfig{},
		prefix:         []MiddlewareConfig{},
		suffix:         []MiddlewareConfig{},
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

	// 设置大小写敏感
	l.app.SetCaseSensitive(AppConfig.RouterCaseSensitive)

	// 设置上传文件允许的最大尺寸
	MaxMemory = AppConfig.MaxMemoryMB * MB

	// 配置数据库
	l.dbService = registerDBService()

	// 初始化全局session
	err := registerSession()
	if err != nil {
		l.app.Logger().Error("Failed to create GlobalSessions: %v.", err)
	}
	l.app.SetSessions(GlobalSessions)

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

func registerAppConfig() (err error) {
	fname := APPCONFIG_FILE
	appconf, err := config.NewConfig("ini", fname)
	if err == nil {
		trySetAppConfig(appconf.(*config.IniConfigContainer))
		return appconf.SaveConfigFile(fname)
	}

	os.MkdirAll(filepath.Dir(fname), 0777)
	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	f.Close()
	appconf, err = config.NewConfig("ini", fname)
	defaultAppConfig(appconf.(*config.IniConfigContainer))
	return appconf.SaveConfigFile(fname)
}

func registerDBConfig() (err error) {
	fname := DBCONFIG_FILE
	appconf, err := config.NewConfig("ini", fname)
	if err == nil {
		trySetDBConfig(appconf.(*config.IniConfigContainer))
		return appconf.SaveConfigFile(fname)
	}

	os.MkdirAll(filepath.Dir(fname), 0777)
	f, err := os.Create(fname)
	if err != nil {
		panic(err)
	}
	f.Close()
	appconf, err = config.NewConfig("ini", fname)
	defaultDBConfig(appconf.(*config.IniConfigContainer))
	return appconf.SaveConfigFile(fname)
}

func registerSession() (err error) {
	if !AppConfig.Session.Enable {
		return
	}
	conf := map[string]interface{}{
		"cookieName":      AppConfig.Session.CookieName,
		"gclifetime":      AppConfig.Session.GcMaxlifetime,
		"providerConfig":  filepath.ToSlash(AppConfig.Session.ProviderConfig),
		"secure":          AppConfig.Listen.EnableHTTPS,
		"enableSetCookie": AppConfig.Session.EnableSetCookie,
		"domain":          AppConfig.Session.Domain,
		"cookieLifeTime":  AppConfig.Session.CookieLifeTime,
	}
	confBytes, _ := json.Marshal(conf)
	GlobalSessions, err = session.NewManager(AppConfig.Session.Provider, string(confBytes))
	if err != nil {
		return
	}
	go GlobalSessions.GC()
	return
}

// 注册固定的静态文件与目录
func registerStaticRouter() {
	DefLessgo.app.Static("/uploads", UPLOADS_DIR, autoHTMLSuffix())
	DefLessgo.app.Static("/static", STATIC_DIR, filterTemplate(), autoHTMLSuffix())
	DefLessgo.app.Static("/bus", BUSINESS_VIEW_DIR, filterTemplate(), autoHTMLSuffix())
	DefLessgo.app.Static("/sys", SYSTEM_VIEW_DIR, filterTemplate(), autoHTMLSuffix())

	DefLessgo.app.File("/favicon.ico", IMG_DIR+"/favicon.ico")
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

func checkHooks(err error) {
	if err == nil {
		return
	}
	DefLessgo.app.Logger().Fatal("%v", err)
}
