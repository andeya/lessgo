package lessgo

import (
	// "github.com/lessgo/lessgo/config"
	"github.com/lessgo/lessgo/session"
)

// Config is the main struct for BConfig
type Config struct {
	AppName             string //Application name
	RunMode             string //Running Mode: dev | prod
	RouterCaseSensitive bool
	ServerName          string
	RecoverPanic        bool
	CopyRequestBody     bool
	EnableGzip          bool
	MaxMemory           int64
	EnableErrorsShow    bool
	Listen              Listen
	WebConfig           WebConfig
	Log                 LogConfig
	DefaultDBConfig     DBConfig
	ExtendDBConfig      map[string]DBConfig
}

// Listen holds for http and https related config
type Listen struct {
	Graceful      bool // Graceful means use graceful module to start the server
	ServerTimeOut int64
	ListenTCP4    bool
	EnableHTTP    bool
	HTTPAddr      string
	HTTPPort      int
	EnableHTTPS   bool
	HTTPSAddr     string
	HTTPSPort     int
	HTTPSCertFile string
	HTTPSKeyFile  string
	EnableAdmin   bool
	AdminAddr     string
	AdminPort     int
	EnableFcgi    bool
	EnableStdIo   bool // EnableStdIo works with EnableFcgi Use FCGI via standard I/O
}

// WebConfig holds web related config
type WebConfig struct {
	AutoRender             bool
	EnableDocs             bool
	FlashName              string
	FlashSeparator         string
	DirectoryIndex         bool
	StaticDir              map[string]string
	StaticExtensionsToGzip []string
	TemplateLeft           string
	TemplateRight          string
	ViewsPath              string
	EnableXSRF             bool
	XSRFKey                string
	XSRFExpire             int
	Session                SessionConfig
}

// SessionConfig holds session related config
type SessionConfig struct {
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
type LogConfig struct {
	AccessLogs  bool
	FileLineNum bool
	Outputs     map[string]string // Store Adaptor : config
}

// DataBase connection Config
type DBConfig struct {
	DBName     string
	DriverName string // DriverName：mssql | odbc(mssql) | mysql | mymysql | postgres | sqlite3 | oci8 | goracle
	ConnString string
}

const (
	UPLOAD_DIR = "./Uploads"
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
	// appConfigProvider is the provider for the config, default is ini
	appConfigProvider = "ini"
)
