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
		MaxMemory           int64  // 文件上传默认内存缓存大小，默认值是 1 << 26(64M)。
		Listen              Listen
		Session             SessionConfig
		Log                 LogConfig
		DefaultDB           DBConfig
		ExtendDB            map[string]DBConfig
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
	// DataBase connection Config
	DBConfig struct {
		DBName     string
		DriverName string // DriverName：mssql | odbc(mssql) | mysql | mymysql | postgres | sqlite3 | oci8 | goracle
		ConnString string
	}
)

// 项目固定目录文件名称
const (
	BUSINESS_DIR   = "Business"
	SYSTEM_DIR     = "System"
	STATIC_DIR     = "Static"
	UPLOAD_DIR     = "Uploads"
	COMMON_DIR     = "Common"
	MIDDLEWARE_DIR = "Middleware"
	DB_DIR         = "DB"
	CONFIG_DIR     = "Config"
	APP_CONFIG     = "app.config"
	DB_CONFIG      = "db.config"
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
		RouterCaseSensitive: true,
		MaxMemory:           1 << 26,
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
		Log: LogConfig{
			Level:     logs.ERROR,
			AsyncChan: 1000,
		},
		DefaultDB: DBConfig{
			DBName:     "default",
			DriverName: "sqlite3",
			ConnString: COMMON_DIR + "/" + DB_DIR + "/lessgo.db",
		},
	}
}

func defaultConfig(iniconf config.Configer) {
	iniconf.Set("appname", BConfig.AppName)
	iniconf.Set("debug", fmt.Sprint(BConfig.Debug))
	iniconf.Set("casesensitive", fmt.Sprint(BConfig.RouterCaseSensitive))
	iniconf.Set("maxmemory", fmt.Sprint(BConfig.MaxMemory))
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
	iniconf.Set("defaultdb::dbname", fmt.Sprint(BConfig.DefaultDB.DBName))
	iniconf.Set("defaultdb::driver", fmt.Sprint(BConfig.DefaultDB.DriverName))
	iniconf.Set("defaultdb::connstring", fmt.Sprint(BConfig.DefaultDB.ConnString))
}
func trySet(iniconf config.Configer) {
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
	if AppConfig.MaxMemory, err = iniconf.Int64("maxmemory"); AppConfig.MaxMemory <= 0 || err != nil {
		iniconf.Set("maxmemory", fmt.Sprint(BConfig.MaxMemory))
		AppConfig.MaxMemory = BConfig.MaxMemory
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
	if AppConfig.DefaultDB.DBName = iniconf.String("defaultdb::dbname"); AppConfig.DefaultDB.DBName == "" {
		iniconf.Set("defaultdb::dbname", fmt.Sprint(BConfig.DefaultDB.DBName))
		AppConfig.DefaultDB.DBName = BConfig.DefaultDB.DBName
	}
	if AppConfig.DefaultDB.DriverName = iniconf.String("defaultdb::driver"); AppConfig.DefaultDB.DriverName == "" {
		iniconf.Set("defaultdb::driver", fmt.Sprint(BConfig.DefaultDB.DriverName))
		AppConfig.DefaultDB.DriverName = BConfig.DefaultDB.DriverName
	}
	if AppConfig.DefaultDB.ConnString = iniconf.String("defaultdb::connstring"); AppConfig.DefaultDB.ConnString == "" {
		iniconf.Set("defaultdb::connstring", fmt.Sprint(BConfig.DefaultDB.ConnString))
		AppConfig.DefaultDB.ConnString = BConfig.DefaultDB.ConnString
	}

	AppConfig.ExtendDB = map[string]DBConfig{}
	for k, v := range iniconf.(*config.IniConfigContainer).GetAllSections() {
		if !strings.HasPrefix(k, "extenddb_") {
			continue
		}
		AppConfig.ExtendDB[k] = DBConfig{
			DBName:     v["dbname"],
			DriverName: v["driver"],
			ConnString: v["connstring"],
		}
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
