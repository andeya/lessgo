package lessgo

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/lessgo/lessgo/config"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
)

type (
	// Config is the main struct for BConfig
	Config struct {
		AppName             string // Application name
		Info                Info   // Application info
		Debug               bool   // enable/disable debug mode.
		CrossDomain         bool
		RouterCaseSensitive bool  // 是否路由忽略大小写匹配，默认是 true，区分大小写
		MaxMemoryMB         int64 // 文件上传默认内存缓存大小，单位MB
		Listen              Listen
		Session             SessionConfig
		Log                 LogConfig
		FileCache           FileCacheConfig
		DefaultDB           string
		DBList              map[string]DBConfig
	}
	Info struct {
		Version           string
		Description       string
		Email             string
		TermsOfServiceUrl string
		License           string
		LicenseUrl        string
	}
	// Listen holds for http and https related config
	Listen struct {
		Graceful      bool // Graceful means use graceful module to start the server
		Address       string
		ReadTimeout   int64
		WriteTimeout  int64
		EnableHTTPS   bool
		HTTPSKeyFile  string
		HTTPSCertFile string
	}
	// SessionConfig holds session related config
	SessionConfig struct {
		SessionOn               bool
		SessionProvider         string
		SessionName             string
		SessionGCMaxLifetime    int64
		SessionProviderConfig   string
		SessionCookieLifeTime   int
		SessionAutoSetCookie    bool
		SessionDomain           string
		EnableSidInHttpHeader   bool //	enable store/get the sessionId into/from http headers
		SessionNameInHttpHeader string
		EnableSidInUrlQuery     bool //	enable get the sessionId from Url Query params
	}

	// LogConfig holds Log related config
	LogConfig struct {
		Level     int
		AsyncChan int64
	}
	FileCacheConfig struct {
		CacheSecond       int64 // 静态资源缓存监测频率与缓存动态释放的最大时长，单位秒，默认600秒
		SingleFileAllowMB int64 // 允许的最大文件，单位MB
		MaxCapMB          int64 // 最大缓存总量，单位MB
	}
	// DataBase connection Config
	DBConfig struct {
		Name         string
		Driver       string // Driver：mssql | odbc(mssql) | mysql | mymysql | postgres | sqlite3 | oci8 | goracle
		ConnString   string
		MaxOpenConns int
		MaxIdleConns int
		TableFix     string // 表命名空间是前缀还是后缀：prefix | suffix
		TableSpace   string // 表命名空间
		TableSnake   bool   // 表名使用snake风格或保持不变
		ColumnFix    string // 列命名空间是前缀还是后缀：prefix | suffix
		ColumnSpace  string // 列命名空间
		ColumnSnake  bool   // 列名使用snake风格或保持不变
		DisableCache bool
		ShowExecTime bool
		ShowSql      bool
	}
)

// 项目固定目录文件名称
const (
	BUSINESS_API_DIR  = "BusinessApi"
	BUSINESS_VIEW_DIR = "BusinessView"
	SYSTEM_API_DIR    = "SystemApi"
	SYSTEM_VIEW_DIR   = "SystemView"
	STATIC_DIR        = "Static"
	IMG_DIR           = STATIC_DIR + "/Img"
	JS_DIR            = STATIC_DIR + "/Js"
	CSS_DIR           = STATIC_DIR + "/Css"
	TPL_DIR           = STATIC_DIR + "/Tpl"
	PLUGIN_DIR        = STATIC_DIR + "/Plugin"
	UPLOADS_DIR       = "Uploads"
	COMMON_DIR        = "Common"
	MIDDLEWARE_DIR    = COMMON_DIR + "/Middleware"

	TPL_EXT         = ".tpl"
	STATIC_HTML_EXT = ".html"

	CONFIG_DIR     = "Config"
	APPCONFIG_FILE = CONFIG_DIR + "/app.config"
	DBCONFIG_FILE  = CONFIG_DIR + "/db.config"

	DB_DIR            = COMMON_DIR + "/DB"
	DEFAULTDB_SECTION = "defaultdb"

	VIEW_PKG      = "/View"
	MODULE_SUFFIX = "Module"
)

var (
	// AppConfig is the instance of Config, store the config information from file
	AppConfig = initConfig()
	// GlobalSessions is the instance for the session manager
	GlobalSessions *session.Manager
)

func initConfig() *Config {
	return &Config{
		AppName: "lessgo",
		Info: Info{
			Version:     "0.4.0",
			Description: "A simple, stable, efficient and flexible web framework.",
			// Host:              "127.0.0.1:8080",
			Email:             "henrylee_cn@foxmail.com",
			TermsOfServiceUrl: "https://github.com/lessgo/lessgo",
			License:           "MIT",
			LicenseUrl:        "https://github.com/lessgo/lessgo/raw/master/doc/LICENSE",
		},
		Debug:               true,
		CrossDomain:         false,
		RouterCaseSensitive: false,
		MaxMemoryMB:         64, // 64MB
		Listen: Listen{
			Graceful:      false,
			Address:       "0.0.0.0:8080",
			ReadTimeout:   0,
			WriteTimeout:  0,
			EnableHTTPS:   false,
			HTTPSCertFile: "",
			HTTPSKeyFile:  "",
		},
		Session: SessionConfig{
			SessionOn:               false,
			SessionProvider:         "memory",
			SessionName:             "lessgosessionID",
			SessionGCMaxLifetime:    3600,
			SessionProviderConfig:   "",
			SessionCookieLifeTime:   0, //set cookie default is the browser life
			SessionAutoSetCookie:    true,
			SessionDomain:           "",
			EnableSidInHttpHeader:   false, //	enable store/get the sessionId into/from http headers
			SessionNameInHttpHeader: "Lessgosessionid",
			EnableSidInUrlQuery:     false, //	enable get the sessionId from Url Query params
		},

		FileCache: FileCacheConfig{
			CacheSecond:       600, // 600s
			SingleFileAllowMB: 64,  // 64MB
			MaxCapMB:          256, // 256MB
		},
		Log: LogConfig{
			Level:     logs.DEBUG,
			AsyncChan: 1000,
		},
		DefaultDB: "lessgo",
		DBList: map[string]DBConfig{
			"lessgo": {
				Name:         "lessgo",
				Driver:       "sqlite3",
				ConnString:   DB_DIR + "/sqlite.db",
				MaxOpenConns: 1,
				MaxIdleConns: 1,
				TableFix:     "prefix",
				TableSpace:   "",
				TableSnake:   true,
				ColumnFix:    "prefix",
				ColumnSpace:  "",
				ColumnSnake:  true,
				DisableCache: false,
				ShowExecTime: false,
				ShowSql:      false,
			},
		},
	}
}

func LoadAppConfig() (err error) {
	fname := APPCONFIG_FILE
	iniconf, err := config.NewConfig("ini", fname)
	if err == nil {
		syncSingleConfig("system", AppConfig, iniconf)
		syncSingleConfig("filecache", &AppConfig.FileCache, iniconf)
		syncSingleConfig("info", &AppConfig.Info, iniconf)
		syncSingleConfig("listen", &AppConfig.Listen, iniconf)
		syncSingleConfig("log", &AppConfig.Log, iniconf)
		syncSingleConfig("session", &AppConfig.Session, iniconf)
	} else {
		os.MkdirAll(filepath.Dir(fname), 0777)
		f, err := os.Create(fname)
		if err != nil {
			return err
		}
		f.Close()
		iniconf, err = config.NewConfig("ini", fname)
		if err != nil {
			return err
		}
		initSingleConfig("system", AppConfig, iniconf)
		initSingleConfig("filecache", &AppConfig.FileCache, iniconf)
		initSingleConfig("info", &AppConfig.Info, iniconf)
		initSingleConfig("listen", &AppConfig.Listen, iniconf)
		initSingleConfig("log", &AppConfig.Log, iniconf)
		initSingleConfig("session", &AppConfig.Session, iniconf)
	}

	return iniconf.SaveConfigFile(fname)
}

func LoadDBConfig() (err error) {
	fname := DBCONFIG_FILE
	iniconf, err := config.NewConfig("ini", fname)
	if err == nil {
		sysDB := AppConfig.DBList["lessgo"]
		defDB := sysDB
		delete(AppConfig.DBList, "lessgo") // 移除预设数据库
		for _, section := range iniconf.(*config.IniConfigContainer).Sections() {
			dbconfig := defDB
			syncSingleConfig(section, &dbconfig, iniconf)
			if strings.ToLower(section) == DEFAULTDB_SECTION {
				AppConfig.DefaultDB = dbconfig.Name
			}
			AppConfig.DBList[dbconfig.Name] = dbconfig
		}
		if _, ok := AppConfig.DBList["lessgo"]; !ok {
			section := "lessgo"
			if AppConfig.DefaultDB == "lessgo" {
				section = DEFAULTDB_SECTION
			}
			initSingleConfig(section, &sysDB, iniconf)
		}
	} else {
		os.MkdirAll(filepath.Dir(fname), 0777)
		f, err := os.Create(fname)
		if err != nil {
			return err
		}
		f.Close()
		iniconf, err = config.NewConfig("ini", fname)
		if err != nil {
			return err
		}
		for _, dbconfig := range AppConfig.DBList {
			if AppConfig.DefaultDB == dbconfig.Name {
				initSingleConfig(DEFAULTDB_SECTION, &dbconfig, iniconf)
			} else {
				initSingleConfig(dbconfig.Name, &dbconfig, iniconf)
			}
		}
	}

	return iniconf.SaveConfigFile(fname)
}

func syncSingleConfig(section string, p interface{}, iniconf config.Configer) {
	pt := reflect.TypeOf(p)
	if pt.Kind() != reflect.Ptr {
		return
	}
	pt = pt.Elem()
	if pt.Kind() != reflect.Struct {
		return
	}
	pv := reflect.ValueOf(p).Elem()

	for i := 0; i < pt.NumField(); i++ {
		pf := pv.Field(i)
		if !pf.CanSet() {
			continue
		}
		name := pt.Field(i).Name
		fullname := getfullname(section, name)
		switch pf.Kind() {
		case reflect.String:
			str := iniconf.DefaultString(fullname, pf.String())
			switch name {
			case "TableFix", "ColumnFix":
				pf.SetString(strings.ToLower(str))
			default:
				pf.SetString(str)
			}

		case reflect.Int, reflect.Int64:
			num := int64(iniconf.DefaultInt64(fullname, pf.Int()))
			switch fullname {
			case "system::maxmemorymb",
				"filecache::cachesecond", "filecache::singlefileallowmb", "filecache::maxcapmb",
				"listen::readtimeout", "listen::writetimeout",
				"session::sessiongcmaxlifetime", "session::sessioncookielifetime",
				"log::asyncchan":
				if num > 0 {
					pf.SetInt(num)
				}
			case "log::level":
				str := iniconf.DefaultString(fullname, logLevelString(int(num)))
				num = int64(logLevelInt(str))
				if num != -10 {
					pf.SetInt(num)
				} else {
					iniconf.Set(fullname, str)
				}
				continue
			default:
				pf.SetInt(num)
			}

		case reflect.Bool:
			pf.SetBool(iniconf.DefaultBool(fullname, pf.Bool()))

		default:
			continue
		}
		iniconf.Set(fullname, fmt.Sprint(pf.Interface()))
	}
}

func initSingleConfig(section string, p interface{}, iniconf config.Configer) {
	pt := reflect.TypeOf(p)
	if pt.Kind() != reflect.Ptr {
		return
	}
	pt = pt.Elem()
	if pt.Kind() != reflect.Struct {
		return
	}
	pv := reflect.ValueOf(p).Elem()

	for i := 0; i < pt.NumField(); i++ {
		pf := pv.Field(i)
		if !pf.CanSet() {
			continue
		}
		fullname := getfullname(section, pt.Field(i).Name)
		switch pf.Kind() {
		case reflect.String, reflect.Int, reflect.Int64, reflect.Bool:
			switch fullname {
			case "log::level":
				iniconf.Set(fullname, logLevelString(int(pf.Int())))
			default:
				iniconf.Set(fullname, fmt.Sprint(pf.Interface()))
			}
		}
	}
}

// section name and key name case insensitive
func getfullname(section, name string) string {
	if section == "" {
		return strings.ToLower(name)
	}
	return strings.ToLower(section + "::" + name)
}

func logLevelInt(l string) int {
	switch strings.ToLower(l) {
	case "debug":
		return logs.DEBUG
	case "info":
		return logs.INFO
	case "warn":
		return logs.WARN
	case "error":
		return logs.ERROR
	case "fatal":
		return logs.FATAL
	case "off":
		return logs.OFF
	}
	return -10
}

func logLevelString(l int) string {
	switch l {
	case logs.DEBUG:
		return "debug"
	case logs.INFO:
		return "info"
	case logs.WARN:
		return "warn"
	case logs.ERROR:
		return "error"
	case logs.FATAL:
		return "fatal"
	case logs.OFF:
		return "off"
	}
	return "error"
}
