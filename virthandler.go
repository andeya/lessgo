package lessgo

import (
	"path"
	"reflect"
	"runtime"
	"strings"
	"sync"
)

// 虚拟操作
type VirtHandler struct {
	id          string            // 操作的唯一标识符
	methods     []string          // 方法列表
	prefix      string            // 路由节点的url前缀(或含参数)
	prefixPath  string            // 路由节点的url前缀的固定路径部分
	prefixParam string            // 路由节点的url前缀的参数部分
	description string            // 描述
	success     string            // 成功后返回的内容描述
	failure     string            // 失败后返回的内容描述
	param       map[string]string // 参数描述
	lock        sync.Mutex
}

var (
	// 防止VirtHandler的id重复
	virtHandlerMap  = map[string]*VirtHandler{}
	virtHandlerLock sync.RWMutex
)

func GetVirtHandler(id string) (*VirtHandler, bool) {
	virtHandlerLock.RLock()
	defer virtHandlerLock.RUnlock()
	vh, ok := virtHandlerMap[id]
	return vh, ok
}

// 创建全局唯一、完整的VirtHandler
func NewVirtHandler(
	handlerfunc HandlerFunc,
	prefix string,
	methods []string,
	description, success, failure string,
	param map[string]string,

) *VirtHandler {
	prefix, prefixPath, prefixParam := creatPrefix(prefix)
	id := handleWareUri(handlerfunc, methods, prefix)
	v := &VirtHandler{
		id:          id,
		methods:     methods,
		prefix:      prefix,
		prefixPath:  prefixPath,
		prefixParam: prefixParam,
		description: description,
		success:     success,
		failure:     failure,
		param:       param,
	}
	if hasVirtHandler(id) {
		return virtHandlerMap[id]
	}
	setVirtHandler(v)
	setHandlerMap(id, handlerfunc)
	return v
}

// 返回虚拟操作列表的副本
func (v *VirtHandler) Methods() []string {
	ms := make([]string, len(v.methods))
	copy(ms, v.methods)
	return ms
}

// 虚拟操作的id
func (v *VirtHandler) Id() string {
	return v.id
}

// 操作的url前缀
func (v *VirtHandler) Prefix() string {
	return v.prefix
}

// 操作的url前缀的固定路径部分
func (v *VirtHandler) PrefixPath() string {
	return v.prefixPath
}

// 操作的url前缀的参数部分
func (v *VirtHandler) PrefixParam() string {
	return v.prefixParam
}

// 操作的描述
func (v *VirtHandler) Description() string {
	return v.description
}

// 操作成功后返回的内容描述
func (v *VirtHandler) Success() string {
	return v.success
}

// 操作失败后返回的内容描述
func (v *VirtHandler) Failure() string {
	return v.failure
}

// 操作的参数描述的副本
func (v *VirtHandler) Param() map[string]string {
	p := make(map[string]string, len(v.param))
	for key, val := range v.param {
		p[key] = val
	}
	return p
}

func setVirtHandler(vh *VirtHandler) {
	virtHandlerLock.Lock()
	defer virtHandlerLock.Unlock()
	virtHandlerMap[vh.id] = vh
}

func hasVirtHandler(id string) bool {
	virtHandlerLock.RLock()
	defer virtHandlerLock.RUnlock()
	_, ok := virtHandlerMap[id]
	return ok
}

// 清洗并拆分固定路径与参数
func creatPrefix(prefix string) (cleanPrefix, prefixPath, prefixParam string) {
	cleanPrefix = path.Clean(path.Join("/", prefix))
	cleanPrefix = strings.Split(cleanPrefix, "?")[0]
	s := strings.Split(cleanPrefix, "/:")
	prefixPath = s[0]
	if len(s) > 1 {
		prefixParam = "/:" + strings.Join(s[1:], "/:")
	}
	return
}

func handleWareUri(hw interface{}, methods []string, prefix string) string {
	add := "[" + prefix + "]"
	for _, m := range methods {
		add += "[" + m + "]"
	}
	t := reflect.ValueOf(hw).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(hw).Pointer()).Name() + add
	}
	return t.String() + add
}

// 全部handler及其id
var handlerMap = map[string]HandlerFunc{}

func getHandlerMap(id string) HandlerFunc {
	return handlerMap[id]
}

func setHandlerMap(id string, handler HandlerFunc) {
	handlerMap[id] = handler
}
