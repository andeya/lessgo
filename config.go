package lessgo

import (
	"fmt"
	"strings"

	"github.com/lessgo/lessgo/config"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
)

type (
	// Config is the main struct for BConfig
	Config struct {
		AppName             string // Application name
		Debug               bool   // enable/disable debug mode.
		RouterCaseSensitive bool   // 是否路由忽略大小写匹配，默认是 true，区分大小写
		MaxMemoryMB         int64  // 文件上传默认内存缓存大小，单位MB
		Listen              Listen
		Session             SessionConfig
		Log                 LogConfig
		FileCache           FileCacheConfig
		DefaultDB           string
		DBList              map[string]DBConfig
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
		Enable          bool
		CookieName      string
		Provider        string
		ProviderConfig  string
		GcMaxlifetime   int64
		CookieLifeTime  int64
		EnableSetCookie bool
		Domain          string
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
	// BConfig is the default config for Application
	BConfig = initConfig()
	// AppConfig is the instance of Config, store the config information from file
	AppConfig = initConfig()
	// GlobalSessions is the instance for the session manager
	GlobalSessions *session.Manager
)

func initConfig() *Config {
	return &Config{
		AppName:             "lessgo",
		Debug:               true,
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
			Enable:          false,
			CookieName:      "lessgosessionID",
			Provider:        "memory",
			ProviderConfig:  `{"cookieName":"gosessionid", "enableSetCookie,omitempty": true, "gclifetime":3600, "maxLifetime": 3600, "secure": false, "sessionIDHashFunc": "sha1", "sessionIDHashKey": "", "cookieLifeTime": 3600, "providerConfig": ""}`,
			GcMaxlifetime:   3600,
			CookieLifeTime:  3600,
			EnableSetCookie: true,
			Domain:          "",
		},
		FileCache: FileCacheConfig{
			CacheSecond:       600, // 600s
			SingleFileAllowMB: 64,  // 64MB
			MaxCapMB:          256, // 256MB
		},
		Log: LogConfig{
			Level:     logs.ERROR,
			AsyncChan: 1000,
		},
		DefaultDB: "preset",
		DBList: map[string]DBConfig{
			"preset": {
				Name:         "preset",
				Driver:       "sqlite3",
				ConnString:   DB_DIR + "/sqlite.db",
				MaxOpenConns: 1,
				MaxIdleConns: 1,
			},
		},
	}
}

func defaultAppConfig(iniconf *config.IniConfigContainer) {
	iniconf.Set("appname", BConfig.AppName)
	iniconf.Set("debug", fmt.Sprint(BConfig.Debug))
	iniconf.Set("casesensitive", fmt.Sprint(BConfig.RouterCaseSensitive))
	iniconf.Set("filecache::cachesecond", fmt.Sprint(BConfig.FileCache.CacheSecond))
	iniconf.Set("filecache::singlefileallowmb", fmt.Sprint(BConfig.FileCache.SingleFileAllowMB))
	iniconf.Set("filecache::maxcapmb", fmt.Sprint(BConfig.FileCache.MaxCapMB))
	iniconf.Set("maxmemorymb", fmt.Sprint(BConfig.MaxMemoryMB))
	iniconf.Set("listen::graceful", fmt.Sprint(BConfig.Listen.Graceful))
	iniconf.Set("listen::address", fmt.Sprint(BConfig.Listen.Address))
	iniconf.Set("listen::readtimeout", fmt.Sprint(BConfig.Listen.ReadTimeout))
	iniconf.Set("listen::writetimeout", fmt.Sprint(BConfig.Listen.WriteTimeout))
	iniconf.Set("listen::enablehttps", fmt.Sprint(BConfig.Listen.EnableHTTPS))
	iniconf.Set("listen::httpscertfile", fmt.Sprint(BConfig.Listen.HTTPSCertFile))
	iniconf.Set("listen::httpskeyfile", fmt.Sprint(BConfig.Listen.HTTPSKeyFile))
	iniconf.Set("session::enable", fmt.Sprint(BConfig.Session.Enable))
	iniconf.Set("session::cookiename", fmt.Sprint(BConfig.Session.CookieName))
	iniconf.Set("session::provider", fmt.Sprint(BConfig.Session.Provider))
	iniconf.Set("session::providerconfig", fmt.Sprint(BConfig.Session.ProviderConfig))
	iniconf.Set("session::gcmaxlifetime", fmt.Sprint(BConfig.Session.GcMaxlifetime))
	iniconf.Set("session::cookielifetime", fmt.Sprint(BConfig.Session.CookieLifeTime))
	iniconf.Set("session::enablesetcookie", fmt.Sprint(BConfig.Session.EnableSetCookie))
	iniconf.Set("session::domain", fmt.Sprint(BConfig.Session.Domain))
	iniconf.Set("log::level", logLevelString(BConfig.Log.Level))
	iniconf.Set("log::asyncchan", fmt.Sprint(BConfig.Log.AsyncChan))
}

func defaultDBConfig(iniconf *config.IniConfigContainer) {
	for _, db := range BConfig.DBList {
		var section string
		if BConfig.DefaultDB == db.Name {
			section = fmt.Sprintf("%v::", DEFAULTDB_SECTION)
		} else {
			section = fmt.Sprintf("%v::", db.Name)
		}
		iniconf.Set(section+"name", db.Name)
		iniconf.Set(section+"driver", db.Driver)
		iniconf.Set(section+"connstring", db.ConnString)
		iniconf.Set(section+"maxopenconns", fmt.Sprint(db.MaxOpenConns))
		iniconf.Set(section+"maxidleconns", fmt.Sprint(db.MaxIdleConns))
	}
}

func trySetAppConfig(iniconf *config.IniConfigContainer) {
	var err error
	if AppConfig.AppName = iniconf.String("appname"); AppConfig.AppName == "" {
		iniconf.Set("appname", BConfig.AppName)
		AppConfig.AppName = BConfig.AppName
	}
	if AppConfig.Debug, err = iniconf.Bool("debug"); err != nil {
		iniconf.Set("debug", fmt.Sprint(BConfig.Debug))
		AppConfig.Debug = BConfig.Debug
	}
	if AppConfig.RouterCaseSensitive, err = iniconf.Bool("casesensitive"); err != nil {
		iniconf.Set("casesensitive", fmt.Sprint(BConfig.RouterCaseSensitive))
		AppConfig.RouterCaseSensitive = BConfig.RouterCaseSensitive
	}
	if AppConfig.FileCache.CacheSecond, err = iniconf.Int64("filecache::cachesecond"); AppConfig.FileCache.CacheSecond <= 0 || err != nil {
		iniconf.Set("filecache::cachesecond", fmt.Sprint(BConfig.FileCache.CacheSecond))
		AppConfig.FileCache.CacheSecond = BConfig.FileCache.CacheSecond
	}
	if AppConfig.FileCache.SingleFileAllowMB, err = iniconf.Int64("filecache::singlefileallowmb"); AppConfig.FileCache.SingleFileAllowMB <= 0 || err != nil {
		iniconf.Set("filecache::singlefileallowmb", fmt.Sprint(BConfig.FileCache.SingleFileAllowMB))
		AppConfig.FileCache.SingleFileAllowMB = BConfig.FileCache.SingleFileAllowMB
	}
	if AppConfig.FileCache.MaxCapMB, err = iniconf.Int64("filecache::maxcapmb"); AppConfig.FileCache.MaxCapMB <= 0 || err != nil {
		iniconf.Set("filecache::maxcapmb", fmt.Sprint(BConfig.FileCache.MaxCapMB))
		AppConfig.FileCache.MaxCapMB = BConfig.FileCache.MaxCapMB
	}
	if AppConfig.MaxMemoryMB, err = iniconf.Int64("maxmemorymb"); AppConfig.MaxMemoryMB <= 0 || err != nil {
		iniconf.Set("maxmemorymb", fmt.Sprint(BConfig.MaxMemoryMB))
		AppConfig.MaxMemoryMB = BConfig.MaxMemoryMB
	}
	if AppConfig.Listen.Graceful, err = iniconf.Bool("listen::graceful"); err != nil {
		iniconf.Set("listen::graceful", fmt.Sprint(BConfig.Listen.Graceful))
		AppConfig.Listen.Graceful = BConfig.Listen.Graceful
	}
	if AppConfig.Listen.Address = iniconf.String("listen::address"); AppConfig.Listen.Address == "" {
		iniconf.Set("listen::address", fmt.Sprint(BConfig.Listen.Address))
		AppConfig.Listen.Address = BConfig.Listen.Address
	}
	if AppConfig.Listen.ReadTimeout, err = iniconf.Int64("listen::readtimeout"); AppConfig.Listen.ReadTimeout < 0 || err != nil {
		iniconf.Set("listen::readtimeout", fmt.Sprint(BConfig.Listen.ReadTimeout))
		AppConfig.Listen.ReadTimeout = BConfig.Listen.ReadTimeout
	}
	if AppConfig.Listen.WriteTimeout, err = iniconf.Int64("listen::writetimeout"); AppConfig.Listen.WriteTimeout < 0 || err != nil {
		iniconf.Set("listen::writetimeout", fmt.Sprint(BConfig.Listen.WriteTimeout))
		AppConfig.Listen.WriteTimeout = BConfig.Listen.WriteTimeout
	}
	if AppConfig.Listen.EnableHTTPS, err = iniconf.Bool("listen::enablehttps"); err != nil {
		iniconf.Set("listen::enablehttps", fmt.Sprint(BConfig.Listen.EnableHTTPS))
		AppConfig.Listen.EnableHTTPS = BConfig.Listen.EnableHTTPS
	}
	if AppConfig.Listen.HTTPSCertFile = iniconf.String("listen::httpscertfile"); AppConfig.Listen.HTTPSCertFile == "" {
		iniconf.Set("listen::httpscertfile", fmt.Sprint(BConfig.Listen.HTTPSCertFile))
		AppConfig.Listen.HTTPSCertFile = BConfig.Listen.HTTPSCertFile
	}
	if AppConfig.Listen.HTTPSKeyFile = iniconf.String("listen::httpskeyfile"); AppConfig.Listen.HTTPSKeyFile == "" {
		iniconf.Set("listen::httpskeyfile", fmt.Sprint(BConfig.Listen.HTTPSKeyFile))
		AppConfig.Listen.HTTPSKeyFile = BConfig.Listen.HTTPSKeyFile
	}
	if AppConfig.Session.Enable, err = iniconf.Bool("session::enable"); err != nil {
		iniconf.Set("session::enable", fmt.Sprint(BConfig.Session.Enable))
		AppConfig.Session.Enable = BConfig.Session.Enable
	}
	if AppConfig.Session.CookieName = iniconf.String("session::cookiename"); AppConfig.Session.CookieName == "" {
		iniconf.Set("session::cookiename", fmt.Sprint(BConfig.Session.CookieName))
		AppConfig.Session.CookieName = BConfig.Session.CookieName
	}
	if AppConfig.Session.Provider = iniconf.String("session::provider"); AppConfig.Session.Provider == "" {
		iniconf.Set("session::provider", fmt.Sprint(BConfig.Session.Provider))
		AppConfig.Session.Provider = BConfig.Session.Provider
	}
	if AppConfig.Session.ProviderConfig = iniconf.String("session::providerconfig"); AppConfig.Session.ProviderConfig == "" {
		iniconf.Set("session::providerconfig", fmt.Sprint(BConfig.Session.ProviderConfig))
		AppConfig.Session.ProviderConfig = BConfig.Session.ProviderConfig
	}
	if AppConfig.Session.GcMaxlifetime, err = iniconf.Int64("session::gcmaxlifetime"); AppConfig.Session.GcMaxlifetime < 0 || err != nil {
		iniconf.Set("session::gcmaxlifetime", fmt.Sprint(BConfig.Session.GcMaxlifetime))
		AppConfig.Session.GcMaxlifetime = BConfig.Session.GcMaxlifetime
	}
	if AppConfig.Session.CookieLifeTime, err = iniconf.Int64("session::gcmaxlifetime"); AppConfig.Session.CookieLifeTime < 0 || err != nil {
		iniconf.Set("session::cookielifetime", fmt.Sprint(BConfig.Session.CookieLifeTime))
		AppConfig.Session.CookieLifeTime = BConfig.Session.CookieLifeTime
	}
	if AppConfig.Session.EnableSetCookie, err = iniconf.Bool("session::enablesetcookie"); err != nil {
		iniconf.Set("session::enablesetcookie", fmt.Sprint(BConfig.Session.EnableSetCookie))
		AppConfig.Session.EnableSetCookie = BConfig.Session.EnableSetCookie
	}
	if AppConfig.Session.Domain = iniconf.String("session::domain"); AppConfig.Session.Domain == "" {
		iniconf.Set("session::domain", fmt.Sprint(BConfig.Session.Domain))
		AppConfig.Session.Domain = BConfig.Session.Domain
	}
	if AppConfig.Log.Level = logLevelInt(iniconf.String("log::level")); AppConfig.Log.Level == -10 {
		iniconf.Set("log::level", logLevelString(BConfig.Log.Level))
		AppConfig.Log.Level = BConfig.Log.Level
	}
	if AppConfig.Log.AsyncChan, err = iniconf.Int64("log::asyncchan"); AppConfig.Log.AsyncChan <= 0 || err != nil {
		iniconf.Set("log::asyncchan", fmt.Sprint(BConfig.Log.AsyncChan))
		AppConfig.Log.AsyncChan = BConfig.Log.AsyncChan
	}
}

func trySetDBConfig(iniconf *config.IniConfigContainer) {
	for _, s := range iniconf.Sections() {
		dbconfig := DBConfig{
			Name:         iniconf.String(s + "::name"),
			Driver:       iniconf.String(s + "::driver"),
			ConnString:   iniconf.String(s + "::connstring"),
			MaxOpenConns: iniconf.DefaultInt(s+"::maxopenconns", 1),
			MaxIdleConns: iniconf.DefaultInt(s+"::maxidleconns", 1),
		}
		if strings.ToLower(s) == DEFAULTDB_SECTION {
			AppConfig.DefaultDB = dbconfig.Name
		}
		AppConfig.DBList[dbconfig.Name] = dbconfig
	}
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
