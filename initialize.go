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
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/utils"
)

func newLessgo() *Lessgo {
	printInfo()
	err := Config.LoadMainConfig(APPCONFIG_FILE)
	if err != nil {
		fmt.Println(err)
	}
	err = Config.LoadDBConfig(DBCONFIG_FILE)
	if err != nil {
		fmt.Println(err)
	}
	registerMime()

	l := &Lessgo{
		App:            newApp(),
		config:         Config,
		home:           "/",
		serverEnable:   true,
		apiHandlers:    []*ApiHandler{},
		apiMiddlewares: []*ApiMiddleware{},
		before:         []*MiddlewareConfig{},
		after:          []*MiddlewareConfig{},
		virtRouter:     newRootVirtRouter(),
	}

	// 初始化全局日志
	Log.SetMsgChan(Config.Log.AsyncChan)
	Log.SetLevel(Config.Log.Level)

	// 设置运行模式
	l.App.SetDebug(Config.Debug)

	// 设置静态资源缓存
	l.App.SetMemoryCache(NewMemoryCache(
		Config.FileCache.SingleFileAllowMB*MB,
		Config.FileCache.MaxCapMB*MB,
		time.Duration(Config.FileCache.CacheSecond)*time.Second),
	)

	// 设置渲染接口
	l.App.SetRenderer(NewPongo2Render(Config.Debug))

	// 设置上传文件允许的最大尺寸
	MaxMemory = Config.MaxMemoryMB * MB

	// 配置数据库
	l.dbService = registerDBService()

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

// 添加系统预设的中间件
func registerMiddleware() {
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

// 添加系统预设的静态虚拟路由
func registerStaticRouter() {
	File("/favicon.ico", IMG_DIR+"/favicon.ico")
	Static("/uploads", UPLOADS_DIR, &MiddlewareConfig{Name: "智能追加.html后缀"})
	Static("/static", STATIC_DIR, &MiddlewareConfig{Name: "过滤前端模板"}, &MiddlewareConfig{Name: "智能追加.html后缀"})
	Static("/biz", BIZ_VIEW_DIR, &MiddlewareConfig{Name: "过滤前端模板"}, &MiddlewareConfig{Name: "智能追加.html后缀"})
	Static("/sys", SYS_VIEW_DIR, &MiddlewareConfig{Name: "过滤前端模板"}, &MiddlewareConfig{Name: "智能追加.html后缀"})
}

// 注册数据库服务
func registerDBService() *dbservice.DBService {
	dbService := &dbservice.DBService{
		List: map[string]*xorm.Engine{},
	}
	for _, conf := range Config.DBList {
		engine, err := xorm.NewEngine(conf.Driver, conf.ConnString)
		if err != nil {
			Log.Error("%v\n", err)
			continue
		}
		logger := dbservice.NewILogger(Config.Log.AsyncChan, Config.Log.Level, conf.Name)
		logger.BeeLogger.EnableFuncCallDepth(Config.Debug)

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
				Log.Error("%v", err)
			} else {
				f.Close()
			}
		}

		dbService.List[conf.Name] = engine
		if Config.DefaultDB == conf.Name {
			dbService.Default = engine
		}
	}
	return dbService
}
