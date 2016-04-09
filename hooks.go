package lessgo

import (
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"

	"github.com/go-xorm/core"
	"github.com/go-xorm/xorm"

	"github.com/lessgo/lessgo/config"
	"github.com/lessgo/lessgo/dbservice"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
)

func printInfo() {
	fmt.Printf(">%s %s (%s)\n", NAME, VERSION, ADDRESS)
}

func registerMime() error {
	for k, v := range mimemaps {
		mime.AddExtensionType(k, v)
	}
	return nil
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

func registerRouter() error {
	// 从数据读取动态配置

	// 与源码配置进行同步

	// 创建真实路由
	ResetRealRoute()

	return nil
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
	confBytes, err := json.Marshal(conf)
	if err != nil {
		return err
	}
	sessionConfig := string(confBytes)
	GlobalSessions, err = session.NewManager(AppConfig.Session.Provider, sessionConfig)
	if err != nil {
		return
	}
	go GlobalSessions.GC()
	return
}

func checkHooks(err error) {
	if err == nil {
		return
	}
	DefLessgo.Echo.Logger().Fatal("%v", err)
}

func newDBAccess() *dbservice.DBAccess {
	access := &dbservice.DBAccess{
		List: map[string]*xorm.Engine{},
	}
	for _, conf := range AppConfig.DBList {
		engine, err := xorm.NewEngine(conf.Driver, conf.ConnString)
		if err != nil {
			logs.Error("%v", err)
			continue
		}
		logger := dbservice.NewILogger(AppConfig.Log.AsyncChan, AppConfig.Log.Level, conf.Name)
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

		access.List[conf.Name] = engine
		if AppConfig.DefaultDB == conf.Name {
			access.Default = engine
		}
	}
	return access
}
