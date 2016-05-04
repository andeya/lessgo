package lessgo

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/logs/color"
)

// 一旦注册，不可再更改
type (
	ApiMiddleware struct {
		Name          string // 全局唯一
		Desc          string
		DefaultConfig interface{} // 默认配置(JSON格式)
		defaultConfig string      // 默认配置的JSON字符串
		Middleware    func(configJSON string) MiddlewareFunc
		inited        bool // 标记是否已经初始化过
	}
	// 虚拟路由中中间件配置信息
	MiddlewareConfig struct {
		Name   string `json:"name"`   // 全局唯一
		Config string `json:"config"` // JSON格式的配置
	}
)

func (a *ApiMiddleware) CreateMiddlewareFunc(config string) MiddlewareFunc {
	if config == "" {
		return a.Middleware(a.defaultConfig)
	}
	return a.Middleware(config)
}

// 注册中间件
func (a ApiMiddleware) Reg() *ApiMiddleware {
	return a.init()
}

// 初始化中间件，设置id并当Name为空时自动添加Name
func (a *ApiMiddleware) init() *ApiMiddleware {
	if a.inited {
		return getApiMiddleware(a.Name)
	}

	if a.Name == "" {
		v := reflect.ValueOf(a.Middleware)
		a.Name = runtime.FuncForPC(v.Pointer()).Name()
	}

	if m := getApiMiddleware(a.Name); m != nil {
		return m
	}

	if a.DefaultConfig != nil {
		b, _ := json.Marshal(a.DefaultConfig)
		a.defaultConfig = string(b)
	}

	a.inited = true

	setApiMiddleware(a)

	return a
}

var (
	apiMiddlewareMap  = map[string]*ApiMiddleware{}
	apiMiddlewareLock sync.RWMutex
)

func getApiMiddleware(name string) *ApiMiddleware {
	apiMiddlewareLock.RLock()
	defer apiMiddlewareLock.RUnlock()
	return apiMiddlewareMap[name]
}

func setApiMiddleware(vh *ApiMiddleware) {
	apiMiddlewareLock.Lock()
	defer apiMiddlewareLock.Unlock()
	apiMiddlewareMap[vh.Name] = vh
	for i, vh2 := range DefLessgo.apiMiddlewares {
		if vh.Name < vh2.Name {
			list := make([]*ApiMiddleware, len(DefLessgo.apiMiddlewares)+1)
			copy(list, DefLessgo.apiMiddlewares[:i])
			list[i] = vh
			copy(list[i+1:], DefLessgo.apiMiddlewares[i:])
			DefLessgo.apiMiddlewares = list
			return
		}
	}
	DefLessgo.apiMiddlewares = append(DefLessgo.apiMiddlewares, vh)
}

// 检查中间件是否存在
func isExistMiddlewares(middlewareConfigs ...MiddlewareConfig) error {
	var errstring string
	for _, m := range middlewareConfigs {
		_, ok := apiMiddlewareMap[m.Name]
		if !ok {
			errstring += " \"" + m.Name + "\""
		}
	}
	if len(errstring) == 0 {
		return nil
	}
	return fmt.Errorf("Specified below middlewares does not exist: %v\n", errstring)
}

// 根据中间件配置生成中间件
func createMiddlewareFuncs(configs []MiddlewareConfig) []MiddlewareFunc {
	mws := make([]MiddlewareFunc, len(configs))
	for i, mw := range configs {
		mws[i] = apiMiddlewareMap[mw.Name].CreateMiddlewareFunc(mw.Config)
	}
	return mws
}

/*
 * system middleware
 */
func init() {
	(&ApiMiddleware{
		Name:       "检查服务器是否启用",
		Desc:       "检查服务器是否启用",
		Middleware: CheckServer,
	}).init()

	(&ApiMiddleware{
		Name:       "检查是否为访问主页",
		Desc:       "检查是否为访问主页",
		Middleware: CheckHome,
	}).init()

	(&ApiMiddleware{
		Name:       "系统运行日志打印",
		Desc:       "RequestLogger returns a middleware that logs HTTP requests.",
		Middleware: RequestLogger,
	}).init()

	(&ApiMiddleware{
		Name: "捕获运行时恐慌",
		Desc: "Recover returns a middleware which recovers from panics anywhere in the chain and handles the control to the centralized HTTPErrorHandler.",
		DefaultConfig: RecoverConfig{
			StackSize:         4 << 10, // 4 KB
			DisableStackAll:   false,
			DisablePrintStack: false,
		},
		Middleware: Recover,
	}).init()

	(&ApiMiddleware{
		Name:       "设置允许跨域",
		Desc:       "根据配置信息设置允许跨域",
		Middleware: CrossDomain,
	}).init()

	// 系统预设中间件
	PreUse(
		MiddlewareConfig{Name: "检查服务器是否启用"},
		MiddlewareConfig{Name: "检查是否为访问主页"},
		MiddlewareConfig{Name: "系统运行日志打印"},
		MiddlewareConfig{Name: "捕获运行时恐慌"},
		MiddlewareConfig{Name: "设置允许跨域"},
	)
}

// 检查服务器是否启用
func CheckServer(config string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			if !ServerEnable() {
				return c.NoContent(http.StatusServiceUnavailable)
			}
			return next(c)
		}
	}
}

// 检查是否为访问主页
func CheckHome(config string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			if c.Request().URL().Path() == "/" {
				c.Request().URL().SetPath(GetHome())
			}
			return next(c)
		}
	}
}

// RequestLogger returns a middleware that logs HTTP requests.
func RequestLogger(config string) MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {
			req := c.Request()
			res := c.Response()

			remoteAddr := req.RemoteAddress()
			if ip := req.Header().Get(HeaderXRealIP); ip != "" {
				remoteAddr = ip
			} else if ip = req.Header().Get(HeaderXForwardedFor); ip != "" {
				remoteAddr = ip
			} else {
				remoteAddr, _, _ = net.SplitHostPort(remoteAddr)
			}

			start := time.Now()
			if err := next(c); err != nil {
				c.Error(err)
			}
			stop := time.Now()
			method := req.Method()
			path := req.URL().Path()
			if path == "" {
				path = "/"
			}
			size := res.Size()

			n := res.Status()
			code := color.Green(n)
			switch {
			case n >= 500:
				code = color.Red(n)
			case n >= 400:
				code = color.Yellow(n)
			case n >= 300:
				code = color.Cyan(n)
			}

			logs.Debug("%s | %s | %s | %s | %s | %d", remoteAddr, method, path, code, stop.Sub(start), size)
			return nil
		}
	}
}

type (
	// RecoverConfig defines the config for recover middleware.
	RecoverConfig struct {
		// StackSize is the stack size to be printed.
		// Optional with default value as 4k.
		StackSize int

		// DisableStackAll disables formatting stack traces of all other goroutines
		// into buffer after the trace for the current goroutine.
		// Optional with default value as false.
		DisableStackAll bool

		// DisablePrintStack disables printing stack trace.
		// Optional with default value as false.
		DisablePrintStack bool
	}
)

// RecoverWithConfig returns a recover middleware from config.
// See `Recover()`.
func Recover(configJSON string) MiddlewareFunc {
	config := RecoverConfig{}
	json.Unmarshal([]byte(configJSON), &config)

	// Defaults
	if config.StackSize == 0 {
		config.StackSize = 4 << 10
	}

	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			defer func() {
				if r := recover(); r != nil {
					var err error
					switch r := r.(type) {
					case error:
						err = r
					default:
						err = fmt.Errorf("%v", r)
					}
					stack := make([]byte, config.StackSize)
					length := runtime.Stack(stack, !config.DisableStackAll)
					if !config.DisablePrintStack {
						c.Logger().Error("[%s] %s %s", color.Red("PANIC RECOVER"), err, stack[:length])
					}
					c.Error(err)
				}
			}()
			return next(c)
		}
	}
}

// 设置允许跨域
func CrossDomain(config string) MiddlewareFunc {
	return WrapMiddleware(func(c Context) error {
		if AppConfig.CrossDomain {
			c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		}
		return nil
	})
}

func filterTemplate() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {
			ext := path.Ext(c.Request().URL().Path())
			if len(ext) >= 4 && ext[:4] == TPL_EXT {
				return c.NoContent(http.StatusForbidden)
			}
			return next(c)
		}
	}
}

func autoHTMLSuffix() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) (err error) {
			p := c.Request().URL().Path()
			if p[len(p)-1] != '/' {
				ext := path.Ext(p)
				if ext == "" || ext[0] != '.' {
					c.Request().URL().SetPath(strings.TrimSuffix(p, ext) + STATIC_HTML_EXT + ext)
					c.ParamValues()[0] += STATIC_HTML_EXT
				}
			}
			return next(c)
		}
	}
}
