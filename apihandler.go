package lessgo

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
)

type (
	ApiHandler struct {
		Desc    string   // 本操作的描述
		Method  string   // 请求方法，"*"表示除"WS"外全部方法
		methods []string // 真实的请求方法列表(自动转换: "WS"->"GET", "*"->methods)
		Params  []Param  // 参数说明列表，path参数类型的先后顺序与url中保持一致
		// Produces []string            // 支持的响应内容类型，如["application/xml", "application/json"]
		Handler func(Context) error // 操作

		id     string // 操作的唯一标识符
		suffix string // 路由节点的url参数后缀
		inited bool   // 标记是否已经初始化过
		lock   sync.Mutex
	}
	Param struct {
		Name     string      // 参数名
		In       string      // 参数出现位置formData、query、path、body、header
		Required bool        // 是否必填
		Format   interface{} // 参数值示例(至少为相应go基础类型空值)
		Desc     string      // 参数描述
	}
)

var (
	apiHandlerMap  = map[string]*ApiHandler{}
	apiHandlerLock sync.RWMutex
)

func NilApiHandler(desc string) *ApiHandler {
	a := &ApiHandler{
		Desc: desc,
	}
	a.initId()
	a.inited = true
	if getApiHandler(a.id) != nil {
		return apiHandlerMap[a.id]
	}
	apiHandlerLock.Lock()
	defer apiHandlerLock.Unlock()
	apiHandlerMap[a.id] = a
	return a
}

// 注册操作
func (a ApiHandler) Reg() *ApiHandler {
	return a.init()
}

// 初始化并保存在全局唯一的操作列表中
func (a *ApiHandler) init() *ApiHandler {
	a.lock.Lock()
	defer a.lock.Unlock()
	if a.inited {
		return getApiHandler(a.id)
	}
	a.initMethod()
	a.initParamsAndSuffix()
	a.initId()
	a.inited = true
	if h := getApiHandler(a.id); h != nil {
		return h
	}
	setApiHandler(a)
	return a
}

// 虚拟操作的id
func (a *ApiHandler) Id() string {
	return a.id
}

// 操作的url前缀
func (a *ApiHandler) Suffix() string {
	return a.suffix
}

// 真实的请求方法列表(自动转换: "WS"->"GET", "*"->methods)
func (a *ApiHandler) Methods() []string {
	return a.methods
}

func getApiHandler(id string) *ApiHandler {
	apiHandlerLock.RLock()
	defer apiHandlerLock.RUnlock()
	return apiHandlerMap[id]
}

func setApiHandler(vh *ApiHandler) {
	apiHandlerLock.Lock()
	defer apiHandlerLock.Unlock()
	apiHandlerMap[vh.id] = vh
	for i, vh2 := range DefLessgo.apiHandlers {
		if vh.Id() < vh2.Id() {
			list := make([]*ApiHandler, len(DefLessgo.apiHandlers)+1)
			copy(list, DefLessgo.apiHandlers[:i])
			list[i] = vh
			copy(list[i+1:], DefLessgo.apiHandlers[i:])
			DefLessgo.apiHandlers = list
			return
		}
	}
	DefLessgo.apiHandlers = append(DefLessgo.apiHandlers, vh)
}

func (a *ApiHandler) initParamsAndSuffix() {
	a.suffix = ""
	for i, count := 0, len(a.Params); i < count; i++ {
		if a.Params[i].In == "path" {
			a.Params[i].Required = true //path参数不可缺省
			a.suffix += "/:" + a.Params[i].Name
		}
	}
}

func (a *ApiHandler) initMethod() {
	a.Method = strings.ToUpper(a.Method)
	switch a.Method {
	case ANY:
		a.methods = methods[:]
	case WS:
		a.methods = []string{GET}
	case CONNECT, DELETE, GET, HEAD, OPTIONS, PATCH, POST, PUT, TRACE:
		a.methods = []string{a.Method}
	default:
		Logger().Fatal("ApiHandler \"%v\"'s method can't be %v. ", a.Desc, a.Method)
	}
}

func (a *ApiHandler) initId() {
	add := "[" + a.suffix + "][" + a.Desc + "]" + "[" + a.Method + "]"
	v := reflect.ValueOf(a.Handler)
	t := v.Type()
	if t.Kind() == reflect.Func {
		a.id = runtime.FuncForPC(v.Pointer()).Name() + add
	} else {
		a.id = t.String() + add
	}
}
