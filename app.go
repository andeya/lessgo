package lessgo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/lessgo/lessgo/grace"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/logs/color"
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/utils"
	"github.com/lessgo/lessgo/websocket"
)

type (
	// App is the top-level framework instancthis.
	App struct {
		debug          bool
		router         *Router
		routes         map[string]Route
		routerIndex    int
		chainNodes     []MiddlewareFunc
		chainHandler   HandlerFunc
		failureHandler FailureHandlerFunc
		panicStackFunc PanicStackFunc
		sessions       *session.Manager
		binder         Binder
		renderer       Renderer
		memoryCache    *MemoryCache
		ctxPool        sync.Pool
		lock           sync.RWMutex
	}

	// Route contains a handler and information for matching against requests.
	Route struct {
		Method  string
		Path    string
		Handler string
	}

	// HandlerFunc defines a function to server HTTP requests.
	HandlerFunc func(*Context) error

	FailureHandlerFunc func(c *Context, code int, errString string) error

	PanicStackFunc func(rcv interface{}) string

	// MiddlewareFunc defines a function to process middleware.
	MiddlewareFunc func(HandlerFunc) HandlerFunc

	// Validator is the interface that wraps the Validate function.
	Validator interface {
		Validate() error
	}

	// Renderer is the interface that wraps the Render function.
	Renderer interface {
		Render(io.Writer, string, interface{}, *Context) error
	}
)

// HTTP methods
const (
	CONNECT = "CONNECT"
	DELETE  = "DELETE"
	GET     = "GET"
	HEAD    = "HEAD"
	OPTIONS = "OPTIONS"
	PATCH   = "PATCH"
	POST    = "POST"
	PUT     = "PUT"
	TRACE   = "TRACE"

	WS  = "WS" // websocket "GET"
	ANY = "*"  // exclusion of all methods out of "WS"
)

// MIME types
const (
	MIMEApplicationJSON                  = "application/json"
	MIMEApplicationJSONCharsetUTF8       = MIMEApplicationJSON + "; " + charsetUTF8
	MIMEApplicationJavaScript            = "application/javascript"
	MIMEApplicationJavaScriptCharsetUTF8 = MIMEApplicationJavaScript + "; " + charsetUTF8
	MIMEApplicationXML                   = "application/xml"
	MIMEApplicationXMLCharsetUTF8        = MIMEApplicationXML + "; " + charsetUTF8
	MIMEApplicationForm                  = "application/x-www-form-urlencoded"
	MIMEApplicationProtobuf              = "application/protobuf"
	MIMEApplicationMsgpack               = "application/msgpack"
	MIMETextHTML                         = "text/html"
	MIMETextHTMLCharsetUTF8              = MIMETextHTML + "; " + charsetUTF8
	MIMETextPlain                        = "text/plain"
	MIMETextPlainCharsetUTF8             = MIMETextPlain + "; " + charsetUTF8
	MIMEMultipartForm                    = "multipart/form-data"
	MIMEOctetStream                      = "application/octet-stream"
)

const (
	charsetUTF8 = "charset=utf-8"
)

// Headers
const (
	HeaderAcceptEncoding                = "Accept-Encoding"
	HeaderAuthorization                 = "Authorization"
	HeaderContentDisposition            = "Content-Disposition"
	HeaderContentEncoding               = "Content-Encoding"
	HeaderContentLength                 = "Content-Length"
	HeaderContentType                   = "Content-Type"
	HeaderCookie                        = "Cookie"
	HeaderSetCookie                     = "Set-Cookie"
	HeaderIfModifiedSince               = "If-Modified-Since"
	HeaderLastModified                  = "Last-Modified"
	HeaderLocation                      = "Location"
	HeaderUpgrade                       = "Upgrade"
	HeaderVary                          = "Vary"
	HeaderWWWAuthenticate               = "WWW-Authenticate"
	HeaderXForwardedProto               = "X-Forwarded-Proto"
	HeaderXHTTPMethodOverride           = "X-HTTP-Method-Override"
	HeaderXForwardedFor                 = "X-Forwarded-For"
	HeaderXRealIP                       = "X-Real-IP"
	HeaderServer                        = "Server"
	HeaderOrigin                        = "Origin"
	HeaderAccessControlRequestMethod    = "Access-Control-Request-Method"
	HeaderAccessControlRequestHeaders   = "Access-Control-Request-Headers"
	HeaderAccessControlAllowOrigin      = "Access-Control-Allow-Origin"
	HeaderAccessControlAllowMethods     = "Access-Control-Allow-Methods"
	HeaderAccessControlAllowHeaders     = "Access-Control-Allow-Headers"
	HeaderAccessControlAllowCredentials = "Access-Control-Allow-Credentials"
	HeaderAccessControlExposeHeaders    = "Access-Control-Expose-Headers"
	HeaderAccessControlMaxAge           = "Access-Control-Max-Age"

	// Security
	HeaderStrictTransportSecurity = "Strict-Transport-Security"
	HeaderXContentTypeOptions     = "X-Content-Type-Options"
	HeaderXXSSProtection          = "X-XSS-Protection"
	HeaderXFrameOptions           = "X-Frame-Options"
	HeaderContentSecurityPolicy   = "Content-Security-Policy"
	HeaderXCSRFToken              = "X-CSRF-Token"
)

var (
	methods = [...]string{
		CONNECT,
		DELETE,
		GET,
		HEAD,
		OPTIONS,
		PATCH,
		POST,
		PUT,
		TRACE,
	}
)

// Errors
var (
	ErrUnsupportedMediaType        = NewHTTPError(http.StatusUnsupportedMediaType)
	ErrNotFound                    = NewHTTPError(http.StatusNotFound)
	ErrUnauthorized                = NewHTTPError(http.StatusUnauthorized)
	ErrMethodNotAllowed            = NewHTTPError(http.StatusMethodNotAllowed)
	ErrStatusRequestEntityTooLarge = NewHTTPError(http.StatusRequestEntityTooLarge)
	ErrStatusInternalServerError   = NewHTTPError(http.StatusInternalServerError)
	ErrRendererNotRegistered       = errors.New("renderer not registered")
	ErrInvalidRedirectCode         = errors.New("invalid redirect status code")
	ErrCookieNotFound              = errors.New("cookie not found")
)

var (
	// 请求处理链的最末端(最后被调用的空操作)
	chainEndHandler = HandlerFunc(func(c *Context) error {
		return nil
	})

	// 失败状态默认的响应内容
	defaultFailureHandler = func(c *Context, code int, errString string) error {
		statusText := http.StatusText(code)
		if len(errString) > 0 {
			errString = `<br><p><b style="color:red;">[ERROR]</b> <pre>` + errString + `</pre></p>`
		}

		return c.HTML(code, fmt.Sprintf(
			`<html>
<head><title>%d %s</title></head>
<body bgcolor="white">
<center><h1>%d %s</h1></center>
<hr><center>lessgo/%s</center>
%s
</body>
</html>
`,
			code,
			statusText,
			code,
			statusText,
			VERSION,
			errString,
		))
	}

	// 从请求过程中的恐慌获取显示的日志内容
	defaultPanicStackFunc = func(rcv interface{}) string {
		msg := fmt.Sprint(rcv)
		stack := make([]byte, 4<<10) //4KB
		length := runtime.Stack(stack, true)
		errString := msg + "\n\n[STACK]\n" + utils.Bytes2String(stack[:length])
		return errString
	}
)

// New creates an instance of App.
func newApp() *App {
	this := &App{
		chainHandler:   chainEndHandler,
		binder:         &binder{},
		failureHandler: defaultFailureHandler,
		panicStackFunc: defaultPanicStackFunc,
	}

	this.ctxPool.New = func() interface{} {
		return this.newContext(new(Response), new(http.Request))
	}

	this.router = newRouter()

	this.chainNodes = []MiddlewareFunc{this.router.process}
	this.resetChain()

	this.SetDebug(true)
	return this
}

func (this *App) Log() logs.Logger {
	return Log
}

func (this *App) Sessions() *session.Manager {
	return this.sessions
}

// 设置获取请求过程中恐慌Stack信息的函数
func (this *App) SetPanicStackFunc(fn func(rcv interface{}) string) {
	this.panicStackFunc = PanicStackFunc(fn)
}

// 设置失败状态默认的响应操作
func (this *App) SetFailureHandler(fn func(c *Context, code int, errString string) error) {
	this.failureHandler = FailureHandlerFunc(fn)
}

// SetBinder registers a custom binder. It's invoked by `Context#Bind()`.
func (this *App) SetBinder(b Binder) {
	this.binder = b
}

// SetRenderer registers an HTML template renderer. It's invoked by `Context#Render()`.
func (this *App) SetRenderer(r Renderer) {
	this.renderer = r
}

// SetDebug enable/disable debug modthis.
func (this *App) SetDebug(on bool) {
	this.debug = on
	if this.memoryCache != nil {
		this.memoryCache.SetEnable(!on)
	}
	if on {
		Log.SetLevel(logs.DEBUG)
		Log.EnableFuncCallDepth(true)
	} else {
		Log.EnableFuncCallDepth(false)
	}
}

// Debug returns debug mode (enabled or disabled).
func (this *App) Debug() bool {
	return this.debug
}

// 获取文件缓存对象
func (this *App) MemoryCache() *MemoryCache {
	return this.memoryCache
}

// 判断文件缓存是否开启
func (this *App) CanMemoryCache() bool {
	return this.memoryCache != nil && this.memoryCache.Enable()
}

// 返回当前真实注册的路由列表
func (this *App) RealRoutes() []Route {
	count := len(this.routes)
	keys := make([]string, count)
	routes := make([]Route, count)
	m := this.routes
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	for i, k := range keys {
		routes[i] = m[k]
	}
	return routes
}

// ServeHTTP implements `http.Handler` interface, which serves HTTP requests.
func (this *App) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	this.lock.RLock()
	var (
		err error
		c   = this.ctxPool.Get().(*Context)
	)

	defer func() {
		if rcv := recover(); rcv != nil {
			errString := this.panicStackFunc(rcv)
			if !c.response.Committed() {
				err = this.failureHandler(c, 500, errString)
			}
			Log.Error("[%s] %s", color.Red("PANIC RECOVER"), errString)
		}

		if err != nil {
			Log.Error("%s", err.Error())
		}

		c.free()
		this.ctxPool.Put(c)
		this.lock.RUnlock()
	}()

	if err = c.init(rw, req); err != nil {
		Log.Error("%s", err.Error())
		return
	}

	// Execute chain
	if err = this.chainHandler(c); err != nil {
		errString := err.Error()
		if !c.response.Committed() {
			err = this.failureHandler(c, 500, errString)
		}
		Log.Error("%s", errString)
	}
}

// Run starts the HTTP server.
func (this *App) run(address, tlsCertfile, tlsKeyfile string, readTimeout, writeTimeout time.Duration, graceful bool) {
	server := &http.Server{
		Addr:         address,
		Handler:      this,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
	}

	canHttps := tlsCertfile != "" && tlsKeyfile != ""

	var err error
	if !graceful {
		if canHttps {
			err = server.ListenAndServeTLS(tlsCertfile, tlsKeyfile)
		} else {
			err = server.ListenAndServe()
		}

	} else {

		endRunning := make(chan bool, 1)
		graceServer := grace.NewServer(address, server, Log)
		if canHttps {
			go func() {
				time.Sleep(20 * time.Microsecond)
				if err = graceServer.ListenAndServeTLS(tlsCertfile, tlsKeyfile); err != nil {
					err = fmt.Errorf("Grace-ListenAndServeTLS: %v, %d", err, os.Getpid())
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
				}
			}()
		} else {
			go func() {
				// graceServer.Network = "tcp4"
				if err = graceServer.ListenAndServe(); err != nil {
					err = fmt.Errorf("Grace-ListenAndServe: %v, %d", err, os.Getpid())
					time.Sleep(100 * time.Microsecond)
					endRunning <- true
				}
			}()
		}
		<-endRunning
	}

	if err != nil {
		Log.Fatal("%v", err)
		select {}
	}
}

// 设置文件缓存
func (this *App) setMemoryCache(m *MemoryCache) {
	m.SetEnable(!this.debug)
	this.memoryCache = m
}

func (this *App) setSessions(sessions *session.Manager) {
	this.sessions = sessions
}

// prefixUse adds middlewares to the beginning of chain.
func (this *App) prefixUse(middleware ...MiddlewareFunc) {
	this.routerIndex += len(middleware)
	this.chainNodes = append(middleware, this.chainNodes...)
	this.resetChain()
}

// suffixUse adds middlewares to the end of chain.
func (this *App) suffixUse(middleware ...MiddlewareFunc) {
	this.chainNodes = append(this.chainNodes, middleware...)
	this.resetChain()
}

// beforeUse adds middlewares to the chain which is run before router.
func (this *App) beforeUse(middleware ...MiddlewareFunc) {
	chain := make([]MiddlewareFunc, this.routerIndex)
	copy(chain, this.chainNodes[:this.routerIndex])
	chain = append(chain, middleware...)
	this.chainNodes = append(chain, this.chainNodes[this.routerIndex:]...)
	this.routerIndex += len(middleware)
	this.resetChain()
}

// afterUse adds middlewares to the chain which is run after router.
func (this *App) afterUse(middleware ...MiddlewareFunc) {
	chain := make([]MiddlewareFunc, this.routerIndex+1)
	copy(chain, this.chainNodes[:this.routerIndex+1])
	chain = append(chain, middleware...)
	this.chainNodes = append(chain, this.chainNodes[this.routerIndex+1:]...)
	this.resetChain()
}

func (this *App) cleanRouter() {
	this.router.trees = make(map[string]*node)
	this.routes = make(map[string]Route)
	this.chainNodes = []MiddlewareFunc{this.router.process}
	this.routerIndex = 0
	this.chainHandler = chainEndHandler
}
func (this *App) resetChain() {
	this.chainHandler = chainEndHandler
	for i := len(this.chainNodes) - 1; i >= 0; i-- {
		this.chainHandler = this.chainNodes[i](this.chainHandler)
	}
}

// group creates a new router group with prefix and optional group-level middleware.
func (this *App) group(prefix string, middleware ...MiddlewareFunc) (g *Group) {
	g = &Group{prefix: prefix, app: this}
	g.use(middleware...)
	return
}

// static registers a new route with path prefix to serve static files from the
// provided root directory.
func (this *App) static(prefix, root string, middleware ...MiddlewareFunc) {
	this.addwithlog(false, GET, prefix+"/*filepath", func(c *Context) error {
		return c.File(path.Join(root, c.PathParamByIndex(0)))
	}, middleware...)
	Log.Sys("| %-7s | %-30s | %v", GET, prefix+"/*filepath", root)
}

// file registers a new route with path to serve a static filthis.
func (this *App) file(path, file string, middleware ...MiddlewareFunc) {
	this.addwithlog(false, GET, path, HandlerFunc(func(c *Context) error {
		return c.File(file)
	}), middleware...)
	Log.Sys("| %-7s | %-30s | %v", GET, path, file)
}

// match registers a new route for multiple HTTP methods and path with matching
// handler in the router with optional route-level middleware.
func (this *App) match(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	for _, method := range methods {
		switch method {
		case WS:
			this.webSocket(path, handler, middleware...)
		default:
			this.add(method, path, handler, middleware...)
		}
	}
}

// webSocket adds a webSocket route > handler to the router.
func (this *App) webSocket(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	this.addwithlog(false, GET, path, HandlerFunc(func(c *Context) error {
		websocket.Handler(func(ws *websocket.Conn) {
			c.SetWs(ws)
			err := handler(c)
			ws.Close()
			if err != nil {
				Log.Debug("[ws] %s", err.Error())
			}
		}).ServeHTTP(c.response, c.request)
		return nil
	}), middleware...)
	Log.Sys("| %-7s | %-30s | %v", WS, path, handlerName(handler))
}

func (this *App) add(method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	this.addwithlog(true, method, path, handler, middleware...)
}

func (this *App) addwithlog(logprint bool, method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	path = joinpath(path, "")
	name := handlerName(handler)
	// Chain middleware
	h := handler
	for i := len(middleware) - 1; i >= 0; i-- {
		h = middleware[i](h)
	}
	this.router.Handle(method, path, h)

	this.routes[method+path] = Route{
		Method:  method,
		Path:    path,
		Handler: name,
	}

	if logprint {
		Log.Sys("| %-7s | %-30s | %v", method, path, name)
	}
}

// uri generates a uri from handler.
func (this *App) uri(handler HandlerFunc, params ...interface{}) string {
	uri := new(bytes.Buffer)
	ln := len(params)
	n := 0
	name := handlerName(handler)
	for _, r := range this.routes {
		if r.Handler == name {
			for i, l := 0, len(r.Path); i < l; i++ {
				if r.Path[i] == ':' && n < ln {
					for ; i < l && r.Path[i] != '/'; i++ {
					}
					uri.WriteString(fmt.Sprintf("%v", params[n]))
					n++
				}
				if i < l {
					uri.WriteByte(r.Path[i])
				}
			}
			break
		}
	}
	return uri.String()
}

// url is an alias for `uri` function.
func (this *App) url(h HandlerFunc, params ...interface{}) string {
	return this.uri(h, params...)
}

// newContext returns a Context instancthis.
func (this *App) newContext(resp *Response, req *http.Request) *Context {
	return &Context{
		request:        req,
		response:       resp,
		pvalues:        nil,
		pkeys:          nil,
		store:          make(store),
		failureHandler: this.failureHandler,
	}
}

// getContext returns `Context` from the sync.Pool. You must return the context by
// calling `putContext()`.
func (this *App) getContext() *Context {
	return this.ctxPool.Get().(*Context)
}

// putContext returns `Context` instance back to the sync.Pool. You must call it after
// `getContext()`.
func (this *App) putContext(c *Context) {
	this.ctxPool.Put(c)
}

// HTTPError represents an error that occured while handling a request.
type HTTPError struct {
	Code    int
	Message string
}

// NewHTTPError creates a new HTTPError instancthis.
func NewHTTPError(code int, msg ...string) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(msg) > 0 {
		m := msg[0]
		he.Message = m
	}
	return he
}

// Error makes it compatible with `error` interfacthis.
func (this *HTTPError) Error() string {
	return this.Message
}

func wrapMiddlewares(middleware []interface{}) []MiddlewareFunc {
	ms := make([]MiddlewareFunc, len(middleware))
	for i, m := range middleware {
		ms[i] = WrapMiddleware(m)
	}
	return ms
}

func handlerName(h HandlerFunc) string {
	v := reflect.ValueOf(h)
	t := v.Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(v.Pointer()).Name()
	}
	return t.String()
}

func joinpath(prefix, p string) string {
	u := path.Join(prefix, p)
	return path.Clean("/" + strings.Split(u, "?")[0])
}
