package lessgo

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	confpkg "github.com/lessgo/lessgo/config"
	"github.com/lessgo/lessgo/logs"
)

type (
	// Config is the main struct for BConfig
	config struct {
		AppName     string // Application name
		Info        Info   // Application info
		Debug       bool   // enable/disable debug mode.
		CrossDomain bool
		MaxMemoryMB int64 // 文件上传默认内存缓存大小，单位MB
		Listen      Listen
		Session     SessionConfig
		Log         LogConfig
		FileCache   FileCacheConfig
		DefaultDB   string
		DBList      map[string]DBConfig
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
	BIZ_HANDLER_DIR = "bizhandler"
	BIZ_MODEL_DIR   = "bizmodel"
	BIZ_VIEW_DIR    = "bizview"
	SYS_HANDLER_DIR = "syshandler"
	SYS_MODEL_DIR   = "sysmodel"
	SYS_VIEW_DIR    = "sysview"
	STATIC_DIR      = "static"
	IMG_DIR         = STATIC_DIR + "/img"
	JS_DIR          = STATIC_DIR + "/js"
	CSS_DIR         = STATIC_DIR + "/css"
	TPL_DIR         = STATIC_DIR + "/tpl"
	PLUGIN_DIR      = STATIC_DIR + "/plugin"
	UPLOADS_DIR     = "uploads"
	COMMON_DIR      = "common"
	MIDDLEWARE_DIR  = "middleware"
	ROUTER_DIR      = "router"

	TPL_EXT         = ".tpl"
	STATIC_HTML_EXT = ".html"

	CONFIG_DIR     = "config"
	APPCONFIG_FILE = CONFIG_DIR + "/app.config"
	DBCONFIG_FILE  = CONFIG_DIR + "/db.config"

	DATABASE_DIR      = "database"
	DEFAULTDB_SECTION = "defaultdb"
)

func newConfig() *config {
	return &config{
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
		Debug:       true,
		CrossDomain: false,
		MaxMemoryMB: 64, // 64MB
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
				ConnString:   DATABASE_DIR + "/sqlite.db",
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

func (this *config) LoadMainConfig(fname string) (err error) {
	iniconf, err := confpkg.NewConfig("ini", fname)
	if err == nil {
		os.Remove(fname)
		readSingleConfig("system", Config, iniconf)
		readSingleConfig("filecache", &this.FileCache, iniconf)
		readSingleConfig("info", &this.Info, iniconf)
		readSingleConfig("listen", &this.Listen, iniconf)
		readSingleConfig("log", &this.Log, iniconf)
		readSingleConfig("session", &this.Session, iniconf)
	}
	os.MkdirAll(filepath.Dir(fname), 0777)
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	f.Close()
	iniconf, err = confpkg.NewConfig("ini", fname)
	if err != nil {
		return err
	}
	writeSingleConfig("system", Config, iniconf)
	writeSingleConfig("filecache", &this.FileCache, iniconf)
	writeSingleConfig("info", &this.Info, iniconf)
	writeSingleConfig("listen", &this.Listen, iniconf)
	writeSingleConfig("log", &this.Log, iniconf)
	writeSingleConfig("session", &this.Session, iniconf)

	return iniconf.SaveConfigFile(fname)
}

func (this *config) LoadDBConfig(fname string) (err error) {
	iniconf, err := confpkg.NewConfig("ini", fname)
	if err == nil {
		os.Remove(fname)
		sections := iniconf.(*confpkg.IniConfigContainer).Sections()
		if len(sections) > 0 {
			this.DefaultDB = ""
			defDB := this.DBList["lessgo"]
			delete(this.DBList, "lessgo")
			for _, section := range sections {
				dbconfig := defDB
				readSingleConfig(section, &dbconfig, iniconf)
				if strings.ToLower(section) == DEFAULTDB_SECTION {
					this.DefaultDB = dbconfig.Name
				}
				this.DBList[dbconfig.Name] = dbconfig
			}
			if this.DefaultDB == "" {
				this.DefaultDB = iniconf.DefaultString(sections[0]+"::name", defDB.Name)
			}
		}
	}

	os.MkdirAll(filepath.Dir(fname), 0777)
	f, err := os.Create(fname)
	if err != nil {
		return err
	}
	f.Close()
	iniconf, err = confpkg.NewConfig("ini", fname)
	if err != nil {
		return err
	}
	for _, dbconfig := range this.DBList {
		if this.DefaultDB == dbconfig.Name {
			writeSingleConfig(DEFAULTDB_SECTION, &dbconfig, iniconf)
		} else {
			writeSingleConfig(dbconfig.Name, &dbconfig, iniconf)
		}
	}

	return iniconf.SaveConfigFile(fname)
}

func readSingleConfig(section string, p interface{}, iniconf confpkg.Configer) {
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
				str := logLevelString(int(num))
				str2 := iniconf.DefaultString(fullname, str)
				num = int64(logLevelInt(str2))
				if num != -10 {
					pf.SetInt(num)
				}
			default:
				pf.SetInt(num)
			}

		case reflect.Bool:
			pf.SetBool(iniconf.DefaultBool(fullname, pf.Bool()))
		}
	}
}

func writeSingleConfig(section string, p interface{}, iniconf confpkg.Configer) {
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
