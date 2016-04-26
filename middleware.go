package lessgo

import (
	"fmt"
	"net"
	"net/http"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/logs/color"
)

// 一旦注册，不可再更改
type MiddlewareObj struct {
	Name        string // 全局唯一
	Description string
	MiddlewareFunc
}

func middlewareCheck(middlewareNames []string) error {
	var errstring string
	for _, m := range middlewareNames {
		_, ok := DefLessgo.virtMiddlewares[m]
		if !ok {
			errstring += " \"" + m + "\""
		}
	}
	if len(errstring) == 0 {
		return nil
	}
	return fmt.Errorf("Specified below middlewares does not exist: %v\n", errstring)
}

func getMiddlewares(names []string) []MiddlewareFunc {
	mws := make([]MiddlewareFunc, len(names))
	for i, mw := range names {
		mws[i] = DefLessgo.virtMiddlewares[mw].MiddlewareFunc
	}
	return mws
}

/*
 * system middleware
 */

// 检查服务器是否启用
func CheckServer() MiddlewareFunc {
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
func CheckHome() MiddlewareFunc {
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
func RequestLogger() MiddlewareFunc {
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

var (
	// DefaultRecoverConfig is the default recover middleware config.
	DefaultRecoverConfig = RecoverConfig{
		StackSize:         4 << 10, // 4 KB
		DisableStackAll:   false,
		DisablePrintStack: false,
	}
)

// Recover returns a middleware which recovers from panics anywhere in the chain
// and handles the control to the centralized HTTPErrorHandler.
func Recover() MiddlewareFunc {
	return RecoverWithConfig(DefaultRecoverConfig)
}

// RecoverWithConfig returns a recover middleware from config.
// See `Recover()`.
func RecoverWithConfig(config RecoverConfig) MiddlewareFunc {
	// Defaults
	if config.StackSize == 0 {
		config.StackSize = DefaultRecoverConfig.StackSize
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

var allowCrossDomain = map[string]bool{}

func CrossDomain(c Context) error {
	if AppConfig.CrossDomain || allowCrossDomain[c.Path()] {
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")
	}
	return nil
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
