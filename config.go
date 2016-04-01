package lessgo

import (
	// "github.com/lessgo/lessgo/config"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
)

type (
	// Config is the main struct for BConfig
	Config struct {
		AppName             string // Application name
		Debug               bool   // enable/disable debug mode.
		RouterCaseSensitive bool   // 是否路由忽略大小写匹配，默认是 true，区分大小写
		MaxMemory           int    // 文件上传默认内存缓存大小，默认值是 1 << 26(64M)。
		EnableGzip          bool   // 模板内容是否进行 gzip 或者 zlib 压缩，开启后根据用户的 Accept-Encoding 来判断
		Listen              Listen
		WebConfig           WebConfig
		LogConfig           LogConfig
		SessionConfig       SessionConfig
		DefaultDBConfig     DBConfig
		ExtendDBConfig      map[string]DBConfig
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
	// WebConfig holds web related config
	WebConfig struct {
		// DirectoryIndex         bool
		// StaticExtensionsToGzip []string
		// EnableXSRF             bool
		// XSRFKey                string
		// XSRFExpire             int
		Session SessionConfig
	}
	// SessionConfig holds session related config
	SessionConfig struct {
		SessionOn             bool
		SessionProvider       string
		SessionName           string
		SessionGCMaxLifetime  int64
		SessionProviderConfig string
		SessionCookieLifeTime int
		SessionAutoSetCookie  bool
		SessionDomain         string
	}
	// LogConfig holds Log related config
	LogConfig struct {
		FileLineNum bool
		Async       bool
		Level       int
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
	BConfig *Config
	// AppConfig is the instance of Config, store the config information from file
	AppConfig *Config
	// AppPath is the absolute path to the app
	AppPath string
	// GlobalSessions is the instance for the session manager
	GlobalSessions *session.Manager

	// appConfigPath is the path to the config files
	appConfigPath string
)

func init() {
	BConfig = &Config{
		AppName:             "lessgo",
		Debug:               true,
		RouterCaseSensitive: true,
		MaxMemory:           1 << 26,
		EnableGzip:          false,
		Listen: Listen{
			Graceful:      false,
			Address:       "0.0.0.0:8080",
			ReadTimeout:   0,
			WriteTimeout:  0,
			EnableHTTPS:   false,
			HTTPSCertFile: "",
			HTTPSKeyFile:  "",
		},
		WebConfig: WebConfig{
			SessionConfig{
				SessionOn:             false,
				SessionProvider:       "memory",
				SessionName:           "lessgosessionID",
				SessionGCMaxLifetime:  3600,
				SessionProviderConfig: `{"cookieName":"gosessionid", "enableSetCookie,omitempty": true, "gclifetime":3600, "maxLifetime": 3600, "secure": false, "sessionIDHashFunc": "sha1", "sessionIDHashKey": "", "cookieLifeTime": 3600, "providerConfig": ""}`,
				SessionCookieLifeTime: 3600,
				SessionAutoSetCookie:  true,
				SessionDomain:         "",
			},
		},
		LogConfig: LogConfig{
			FileLineNum: true,
			Async:       true,
			Level:       logs.ERROR,
		},
		DefaultDBConfig: DBConfig{
			DBName:     "default",
			DriverName: "sqlite3",
			ConnString: COMMON_DIR + "/" + DB_DIR + "/lessgo.db",
		},
	}
}
