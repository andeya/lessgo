package lessgo

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/lessgo/lessgo/logs/color"
	"github.com/lessgo/lessgo/utils"
)

/*
 * 中间件
 * ApiMiddleware.Middleware 支持的处理函数类型:
 * MiddlewareFunc
 * func(HandlerFunc) HandlerFunc
 * HandlerFunc
 * func(Context) error
 * ConfMiddlewareFunc
 * func(confObject interface{}) MiddlewareFunc
 */
type ApiMiddleware struct {
	Name       string // 全局唯一
	Desc       string
	Params     []Param     // (可选)参数说明列表(应该只声明当前中间件用到的参数)，path参数类型的先后顺序与url中保持一致
	Config     interface{} // 初始配置，若希望使用参数，则Config不能为nil，至少为对应类型的空值
	Middleware interface{} // 处理函数，类型参考上面注释
	id         string      // 允许不同id相同name的中间件注册，但在name末尾追加"(2)"
	dynamic    bool        // 是否可使用运行时动态配置
	configJSON string      // 若可动态配置，则存入当前配置的JSON字符串
	inited     bool        // 标记是否已经初始化过
	lock       sync.RWMutex
}

// 注册中间件
func (a ApiMiddleware) Reg() *ApiMiddleware {
	a.lock.Lock()
	defer a.lock.Unlock()
	return a.init()
}

// 克隆中间件
func (a *ApiMiddleware) Clone() *ApiMiddleware {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return (&ApiMiddleware{
		Name:       a.Name,
		Desc:       a.Desc,
		Params:     a.Params,
		Config:     a.Config,
		Middleware: a.Middleware,
	}).init()
}

// 设置默认配置，重置中间件
func (a *ApiMiddleware) SetConfig(confObject interface{}) *ApiMiddleware {
	a.lock.Lock()
	defer a.lock.Unlock()
	a.Config = confObject
	a.inited = false
	apiMiddlewareLock.Lock()
	delete(apiMiddlewareMap, a.Name)
	apiMiddlewareLock.Unlock()
	return a.init()
}

// 获取JSON字符串格式的中间件配置
func (a *ApiMiddleware) ConfigJSON() string {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.configJSON
}

// 返回中间件配置结构体
func (a *ApiMiddleware) NewMiddlewareConfig() *MiddlewareConfig {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return &MiddlewareConfig{
		Name:          a.Name,
		Config:        a.configJSON,
		apiMiddleware: a,
	}
}

// 初始化中间件，设置id并当Name为空时自动添加Name
func (a *ApiMiddleware) init() *ApiMiddleware {
	// 检查是否重复初始化
	if a.inited {
		return getApiMiddleware(a.Name)
	}
	defer func() {
		a.inited = true
	}()

	// 获取操作函数URI
	v := reflect.ValueOf(a.Middleware)
	funcName := runtime.FuncForPC(v.Pointer()).Name()

	// 格式化验证中间件处理函数类型
	switch m := a.Middleware.(type) {
	case ConfMiddlewareFunc:
	case func(config interface{}) MiddlewareFunc:
		a.Middleware = ConfMiddlewareFunc(m)
	default:
		a.Middleware = WrapMiddleware(m)
		a.Config = nil
	}

	a.dynamic = false
	a.configJSON = ""
	if a.Config != nil {
		b, err := json.MarshalIndent(a.Config, "", "  ")
		if err == nil {
			a.dynamic = true
			a.configJSON = utils.Bytes2String(b)
		}
	}

	if len(a.Name) == 0 {
		if len(a.Desc) > 0 {
			a.Name = a.Desc
		} else {
			a.Name = funcName
		}
	}

	a.id = utils.MakeHash(a.Name + funcName)
	if m := getApiMiddleware(a.Name); m != nil {
		if m.id == a.id {
			return m
		} else {
			a.Name += "(2)"
			a.id = utils.MakeHash(a.Name + funcName)
		}
	}

	setApiMiddleware(a)

	return a
}

// 获取中间件函数，
// 支持动态配置的中间件可传入JSON字节流进行配置。
func (a *ApiMiddleware) regetFunc(configJSONBytes []byte) (MiddlewareFunc, error) {
	a.lock.Lock()
	defer a.lock.Unlock()
	var err error
	if a.dynamic && len(configJSONBytes) > 0 {
		config := utils.NewObjectPtr(a.Config)
		if json.Unmarshal(configJSONBytes, config) == nil {
			if reflect.TypeOf(a.Config).Kind() != reflect.Ptr {
				config = reflect.ValueOf(config).Elem().Interface()
			}
			return a.Middleware.(Middleware).getMiddlewareFunc(config), nil
		}
		err = fmt.Errorf("Middleware \"%s\" uses initial config, because the type of param is error:\ngot format -> %s,\nwant format -> %s.",
			a.Name, utils.Bytes2String(configJSONBytes), a.configJSON)
	}
	return a.Middleware.(Middleware).getMiddlewareFunc(a.Config), err
}

// 是否支持动态配置
func (a *ApiMiddleware) getDynamic() bool {
	a.lock.RLock()
	defer a.lock.RUnlock()
	return a.dynamic
}

// 虚拟路由中中间件配置信息，用于获取中间件函数
type MiddlewareConfig struct {
	Name          string `json:"name"`   // 全局唯一
	Config        string `json:"config"` // JSON格式的配置（可选）
	apiMiddleware *ApiMiddleware
	lock          sync.RWMutex
}

// 获取*ApiMiddleware
func (m *MiddlewareConfig) GetApiMiddleware() *ApiMiddleware {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.apiMiddleware
}

// 获取JSON字符串格式的中间件配置
func (m *MiddlewareConfig) GetConfig() string {
	m.lock.RLock()
	defer m.lock.RUnlock()
	return m.Config
}

// 以JSON字节流格式配置中间件
func (m *MiddlewareConfig) SetConfig(configJSONBytes []byte) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	_, err := m.apiMiddleware.regetFunc(configJSONBytes)
	if err == nil {
		m.Config = utils.Bytes2String(configJSONBytes)
	}
	return err
}

// 检查是否支持动态配置
func (m *MiddlewareConfig) CheckDynamic() bool {
	m.lock.Lock()
	defer m.lock.Unlock()
	bol := m.apiMiddleware.getDynamic()
	if !bol && len(m.Config) > 0 {
		m.Config = ""
	}
	return bol
}

// 检查是否为有效配置
func (m *MiddlewareConfig) CheckValid() bool {
	m.lock.RLock()
	defer m.lock.RUnlock()
	a := getApiMiddleware(m.Name)
	if a != nil {
		if m.apiMiddleware == a || m.apiMiddleware == nil {
			return true
		}
	}
	return false
}

// 获取中间件操作函数
func (m *MiddlewareConfig) middlewareFunc() MiddlewareFunc {
	m.lock.Lock()
	defer m.lock.Unlock()
	err := m.initApiMiddleware()
	if err != nil {
		Log.Error(err.Error())
		return nil
	}
	fn, err := m.apiMiddleware.regetFunc([]byte(m.Config))
	if err != nil {
		Log.Error(err.Error())
	}
	return fn
}

// 初始化设置中间件对象
func (m *MiddlewareConfig) initApiMiddleware() error {
	if m.apiMiddleware == nil {
		m.apiMiddleware = getApiMiddleware(m.Name)
		if m.apiMiddleware == nil {
			return fmt.Errorf("ApiMiddleware %s is not exist.", m.Name)
		}
	}
	return nil
}

type (
	// 中间件接口
	Middleware interface {
		getMiddlewareFunc(confObject interface{}) MiddlewareFunc
	}
	// 支持配置的中间件处理函数，
	// 若接收参数类型为字符串，且默认配置Config不为nil，则支持运行时动态配置。
	ConfMiddlewareFunc func(confObject interface{}) MiddlewareFunc
)

// 不支持配置的中间件函数实现中间件接口
func (m MiddlewareFunc) getMiddlewareFunc(_ interface{}) MiddlewareFunc {
	return m
}

// 支持配置的中间件处理函数实现中间件接口，
// 若接收参数类型为字符串，且默认配置Config不为nil，则支持运行时动态配置。
func (c ConfMiddlewareFunc) getMiddlewareFunc(confObject interface{}) MiddlewareFunc {
	return c(confObject)
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

func setApiMiddleware(a *ApiMiddleware) {
	apiMiddlewareLock.Lock()
	defer apiMiddlewareLock.Unlock()
	apiMiddlewareMap[a.Name] = a
	for i, a2 := range lessgo.apiMiddlewares {
		if a.Name < a2.Name {
			list := make([]*ApiMiddleware, len(lessgo.apiMiddlewares)+1)
			copy(list, lessgo.apiMiddlewares[:i])
			list[i] = a
			copy(list[i+1:], lessgo.apiMiddlewares[i:])
			lessgo.apiMiddlewares = list
			return
		}
	}
	lessgo.apiMiddlewares = append(lessgo.apiMiddlewares, a)
}

// 检查中间件是否存在
func isExistMiddlewares(middlewareConfigs ...*MiddlewareConfig) error {
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
func getMiddlewareFuncs(configs []*MiddlewareConfig) []MiddlewareFunc {
	mws := make([]MiddlewareFunc, len(configs))
	for i, mw := range configs {
		mws[i] = mw.middlewareFunc()
	}
	return mws
}

/*
 * system middleware
 */

var CheckServer = ApiMiddleware{
	Name: "检查服务器是否启用",
	Desc: "检查服务器是否启用",
	Middleware: func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if !ServerEnable() {
				return c.NoContent(http.StatusServiceUnavailable)
			}
			return next(c)
		}
	},
}.Reg()

var CheckHome = ApiMiddleware{
	Name: "检查是否为访问主页",
	Desc: "检查是否为访问主页",
	Middleware: func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			if c.request.URL.Path == "/" {
				c.request.URL.Path = GetHome()
			}
			return next(c)
		}
	},
}.Reg()

var RequestLogger = ApiMiddleware{
	Name: "系统运行日志打印",
	Desc: "RequestLogger returns a middleware that logs HTTP requests.",
	Middleware: func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			var u = c.request.URL.String()
			start := time.Now()
			if err := next(c); err != nil {
				c.Failure(500, err)
			}
			stop := time.Now()

			method := c.request.Method
			if u == "" {
				u = "/"
			}

			n := c.response.Status()
			var code string
			if runtime.GOOS == "linux" {
				code = strconv.Itoa(n)
			} else {
				code = color.Green(n)
				switch {
				case n >= 500:
					code = color.Red(n)
				case n >= 400:
					code = color.Magenta(n)
				case n >= 300:
					code = color.Cyan(n)
				}
			}

			Log.Debug("%15s | %7s | %s | %8d | %10s | %s", c.RealRemoteAddr(), method, code, c.response.Size(), stop.Sub(start), u)
			return nil
		}
	},
}.Reg()

var CrossDomain = ApiMiddleware{
	Name: "设置允许跨域",
	Desc: "根据配置信息设置允许跨域",
	Middleware: func(c *Context) error {
		c.response.Header().Set("Access-Control-Allow-Credentials", "true")
		c.response.Header().Set("Access-Control-Allow-Origin", c.HeaderParam("Origin"))
		// c.response.Header().Set("Access-Control-Allow-Origin", "*")
		return nil
	},
}.Reg()

var FilterTemplate = ApiMiddleware{
	Name: "过滤前端模板",
	Desc: "过滤前端模板，不允许直接访问",
	Middleware: func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			ext := path.Ext(c.request.URL.Path)
			if len(ext) >= 4 && ext[:4] == TPL_EXT {
				return c.NoContent(http.StatusForbidden)
			}
			return next(c)
		}
	},
}.Reg()

var AutoHTMLSuffix = ApiMiddleware{
	Name: "智能追加.html后缀",
	Desc: "静态路由时智能追加\".html\"后缀",
	Middleware: func(next HandlerFunc) HandlerFunc {
		return func(c *Context) error {
			p := c.request.URL.Path
			if p[len(p)-1] != '/' {
				ext := path.Ext(p)
				if ext == "" || ext[0] != '.' {
					c.request.URL.Path = strings.TrimSuffix(p, ext) + STATIC_HTML_EXT + ext
					c.pvalues[0] += STATIC_HTML_EXT
				}
			}
			return next(c)
		}
	},
}.Reg()
