package lessgo

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"
	"reflect"
	"runtime"
	"strings"
	"sync"

	"github.com/lessgo/lessgo/engine"
	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
)

type (
	// Echo is the top-level framework instance.
	Echo struct {
		prefix           string
		middleware       []MiddlewareFunc
		head             HandlerFunc
		pristineHead     HandlerFunc
		maxParam         *int
		notFoundHandler  HandlerFunc
		httpErrorHandler HTTPErrorHandler
		binder           Binder
		renderer         Renderer
		pool             sync.Pool
		debug            bool
		router           *Router
		logger           logs.Logger
		lock             sync.RWMutex
		routerIndex      int
		caseSensitive    bool
		memoryCache      *MemoryCache
		sessions         *session.Manager
	}

	// Route contains a handler and information for matching against requests.
	Route struct {
		Method  string
		Path    string
		Handler string
	}

	// HTTPError represents an error that occured while handling a request.
	HTTPError struct {
		Code    int
		Message string
	}

	// HandlerFunc defines a function to server HTTP requests.
	HandlerFunc func(Context) error

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, Context)

	// Validator is the interface that wraps the Validate function.
	Validator interface {
		Validate() error
	}

	// Renderer is the interface that wraps the Render function.
	Renderer interface {
		Render(io.Writer, string, interface{}, Context) error
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
	ErrRendererNotRegistered       = errors.New("renderer not registered")
	ErrInvalidRedirectCode         = errors.New("invalid redirect status code")
	ErrCookieNotFound              = errors.New("cookie not found")
)

// Error handlers
var (
	notFoundHandler = func(c Context) error {
		return ErrNotFound
	}

	methodNotAllowedHandler = func(c Context) error {
		return ErrMethodNotAllowed
	}

	// 请求最后被调用的空操作
	headHandlerFunc = HandlerFunc(func(c Context) error {
		return nil
	})
)

// New creates an instance of Echo.
func New() (e *Echo) {
	e = &Echo{
		pristineHead:  headHandlerFunc,
		head:          headHandlerFunc,
		maxParam:      new(int),
		logger:        logs.Global,
		binder:        &binder{},
		caseSensitive: true,
	}
	e.pool.New = func() interface{} {
		return e.NewContext(nil, nil)
	}
	e.router = NewRouter(e)
	e.middleware = []MiddlewareFunc{e.router.Process}
	e.chainMiddleware()

	// Defaults
	e.SetHTTPErrorHandler(e.DefaultHTTPErrorHandler)
	e.logger.AddAdapter("console", "")
	e.logger.AddAdapter("file", `{"filename":"Logger/lessgo.log"}`)
	e.SetDebug(true)
	return
}

// NewContext returns a Context instance.
func (e *Echo) NewContext(rq engine.Request, rs engine.Response) Context {
	return &context{
		request:  rq,
		response: rs,
		echo:     e,
		pvalues:  make([]string, *e.maxParam),
		store:    make(store),
		handler:  notFoundHandler,
	}
}

// Router returns router.
func (e *Echo) Router() *Router {
	return e.router
}

// LogFuncCallDepth enable log funcCallDepth.
func (e *Echo) LogFuncCallDepth(b bool) {
	e.logger.EnableFuncCallDepth(b)
}

// SetLogLevel sets the log level for the logger
func (e *Echo) SetLogLevel(l int) {
	e.logger.SetLevel(l)
}

// AddLogger provides a given logger adapter intologs.Logger with config string.
// config need to be correct JSON as string: {"interval":360}.
func (e *Echo) AddLogAdapter(adaptername string, config string) error {
	return e.logger.AddAdapter(adaptername, config)
}

//logs.Logger returns the logger instance.
func (e *Echo) Logger() logs.Logger {
	return e.logger
}

func (e *Echo) SetCaseSensitive(sensitive bool) {
	e.caseSensitive = sensitive
}

func (e *Echo) CaseSensitive() bool {
	return e.caseSensitive
}

func (e *Echo) Sessions() *session.Manager {
	return e.sessions
}

func (e *Echo) SetSessions(sessions *session.Manager) {
	e.sessions = sessions
}

// DefaultHTTPErrorHandler invokes the default HTTP error handler.
func (e *Echo) DefaultHTTPErrorHandler(err error, c Context) {
	code := http.StatusInternalServerError
	msg := http.StatusText(code)
	if he, ok := err.(*HTTPError); ok {
		code = he.Code
		msg = he.Message
	}
	if e.debug {
		msg = err.Error()
	}
	if !c.Response().Committed() {
		c.String(code, msg)
	}
	e.logger.Debug("%v", err)
}

// SetHTTPErrorHandler registers a custom Echo.HTTPErrorHandler.
func (e *Echo) SetHTTPErrorHandler(h HTTPErrorHandler) {
	e.httpErrorHandler = h
}

// SetBinder registers a custom binder. It's invoked by `Context#Bind()`.
func (e *Echo) SetBinder(b Binder) {
	e.binder = b
}

// SetRenderer registers an HTML template renderer. It's invoked by `Context#Render()`.
func (e *Echo) SetRenderer(r Renderer) {
	e.renderer = r
}

// SetDebug enable/disable debug mode.
func (e *Echo) SetDebug(on bool) {
	e.debug = on
	if e.memoryCache != nil {
		e.memoryCache.SetEnable(!on)
	}
	if on {
		e.logger.SetLevel(logs.DEBUG)
		e.logger.EnableFuncCallDepth(true)
	} else {
		e.logger.EnableFuncCallDepth(false)
	}
}

// Debug returns debug mode (enabled or disabled).
func (e *Echo) Debug() bool {
	return e.debug
}

func (e *Echo) MemoryCacheEnable() bool {
	return e.memoryCache != nil && e.memoryCache.Enable()
}

func (e *Echo) SetMemoryCache(m *MemoryCache) {
	m.SetEnable(!e.debug)
	e.memoryCache = m
}

// PreUse adds middlewares to the beginning of chain.
func (e *Echo) PreUse(middleware ...MiddlewareFunc) {
	e.routerIndex += len(middleware)
	e.middleware = append(middleware, e.middleware...)
	e.chainMiddleware()
}

// SufUse adds middlewares to the end of chain.
func (e *Echo) SufUse(middleware ...MiddlewareFunc) {
	e.middleware = append(e.middleware, middleware...)
	e.chainMiddleware()
}

// BeforeUse adds middlewares to the chain which is run before router.
func (e *Echo) BeforeUse(middleware ...MiddlewareFunc) {
	chain := make([]MiddlewareFunc, e.routerIndex)
	copy(chain, e.middleware[:e.routerIndex])
	chain = append(chain, middleware...)
	e.middleware = append(chain, e.middleware[e.routerIndex:]...)
	e.routerIndex += len(middleware)
	e.chainMiddleware()
}

// AfterUse adds middlewares to the chain which is run after router.
func (e *Echo) AfterUse(middleware ...MiddlewareFunc) {
	chain := make([]MiddlewareFunc, e.routerIndex+1)
	copy(chain, e.middleware[:e.routerIndex+1])
	chain = append(chain, middleware...)
	e.middleware = append(chain, e.middleware[e.routerIndex+1:]...)
	e.chainMiddleware()
}

func (e *Echo) chainMiddleware() {
	e.head = e.pristineHead
	for i := len(e.middleware) - 1; i >= 0; i-- {
		e.head = e.middleware[i](e.head)
	}
}

// Connect registers a new CONNECT route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Connect(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(CONNECT, path, h, m...)
}

// Delete registers a new DELETE route for a path with matching handler in the router
// with optional route-level middleware.
func (e *Echo) Delete(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(DELETE, path, h, m...)
}

// Get registers a new GET route for a path with matching handler in the router
// with optional route-level middleware.
func (e *Echo) Get(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(GET, path, h, m...)
}

// Head registers a new HEAD route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Head(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(HEAD, path, h, m...)
}

// Options registers a new OPTIONS route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Options(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(OPTIONS, path, h, m...)
}

// Patch registers a new PATCH route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Patch(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(PATCH, path, h, m...)
}

// Post registers a new POST route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Post(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(POST, path, h, m...)
}

// Put registers a new PUT route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Put(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(PUT, path, h, m...)
}

// Trace registers a new TRACE route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Trace(path string, h HandlerFunc, m ...MiddlewareFunc) {
	e.add(TRACE, path, h, m...)
}

// Any registers a new route for all HTTP methods and path with matching handler
// in the router with optional route-level middleware.
func (e *Echo) Any(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	for _, m := range methods {
		e.add(m, path, handler, middleware...)
	}
}

// Match registers a new route for multiple HTTP methods and path with matching
// handler in the router with optional route-level middleware.
func (e *Echo) Match(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	for _, m := range methods {
		e.add(m, path, handler, middleware...)
	}
}

// Static registers a new route with path prefix to serve static files from the
// provided root directory.
func (e *Echo) Static(prefix, root string, middleware ...MiddlewareFunc) {
	e.addwithlog(false, GET, prefix+"*", func(c Context) error {
		return c.File(path.Join(root, c.P(0))) // Param `_`
	}, middleware...)
	e.logger.Sys("| %-7s | %-30s | %v", GET, prefix+"*", root)
}

// File registers a new route with path to serve a static file.
func (e *Echo) File(path, file string, middleware ...MiddlewareFunc) {
	e.addwithlog(false, GET, path, HandlerFunc(func(c Context) error {
		return c.File(file)
	}), middleware...)
	e.logger.Sys("| %-7s | %-30s | %v", GET, path, file)
}

func (e *Echo) add(method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	e.addwithlog(true, method, path, handler, middleware...)
}

func (e *Echo) addwithlog(logprint bool, method, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	path = joinpath(path, "")
	// not case sensitive
	if !e.caseSensitive {
		path = strings.ToLower(path)
	}
	name := handlerName(handler)
	e.router.Add(method, path, func(c Context) error {
		h := handler
		// Chain middleware
		for i := len(middleware) - 1; i >= 0; i-- {
			h = middleware[i](h)
		}
		return h(c)
	}, e)

	r := Route{
		Method:  method,
		Path:    path,
		Handler: name,
	}
	e.router.routes = append(e.router.routes, r)
	if logprint {
		e.logger.Sys("| %-7s | %-30s | %v", method, path, name)
	}
}

// Group creates a new router group with prefix and optional group-level middleware.
func (e *Echo) Group(prefix string, m ...MiddlewareFunc) (g *Group) {
	g = &Group{prefix: prefix, echo: e}
	g.Use(m...)
	// Allow all requests to reach the group as they might get dropped if router
	// doesn't find a match, making none of the group middleware process.
	for _, method := range methods {
		e.addwithlog(false, method, prefix+"*", func(c Context) error {
			return c.NoContent(http.StatusNotFound)
		}, g.middleware...)
	}
	return
}

// URI generates a URI from handler.
func (e *Echo) URI(handler HandlerFunc, params ...interface{}) string {
	uri := new(bytes.Buffer)
	ln := len(params)
	n := 0
	name := handlerName(handler)
	for _, r := range e.router.routes {
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

// URL is an alias for `URI` function.
func (e *Echo) URL(h HandlerFunc, params ...interface{}) string {
	return e.URI(h, params...)
}

// Routes returns the registered routes.
func (e *Echo) Routes() []Route {
	return e.router.routes
}

// GetContext returns `Context` from the sync.Pool. You must return the context by
// calling `PutContext()`.
func (e *Echo) GetContext() Context {
	return e.pool.Get().(Context)
}

// PutContext returns `Context` instance back to the sync.Pool. You must call it after
// `GetContext()`.
func (e *Echo) PutContext(c Context) {
	e.pool.Put(c)
}

func (e *Echo) ServeHTTP(rq engine.Request, rs engine.Response) {
	e.lock.RLock()
	defer e.lock.RUnlock()
	if !e.caseSensitive {
		rq.URL().SetPath(strings.ToLower(rq.URL().Path()))
	}

	c := e.pool.Get().(*context)
	c.reset(rq, rs)

	// Execute chain
	if err := e.head(c); err != nil {
		e.httpErrorHandler(err, c)
	}
	e.pool.Put(c)
}

// Run starts the HTTP server.
func (e *Echo) Run(s engine.Server) {
	s.SetHandler(e)
	s.SetLogger(e.logger)
	if err := s.Start(); err != nil {
		e.logger.Fatal("%v", err)
		select {}
	}
}

// NewHTTPError creates a new HTTPError instance.
func NewHTTPError(code int, msg ...string) *HTTPError {
	he := &HTTPError{Code: code, Message: http.StatusText(code)}
	if len(msg) > 0 {
		m := msg[0]
		he.Message = m
	}
	return he
}

// Error makes it compatible with `error` interface.
func (e *HTTPError) Error() string {
	return e.Message
}

// WrapMiddleware wrap `echo.HandlerFunc` into `echo.MiddlewareFunc`.
func WrapMiddleware(h interface{}) MiddlewareFunc {
	var x HandlerFunc
	switch t := h.(type) {
	case MiddlewareFunc:
		return t
	case func(HandlerFunc) HandlerFunc:
		return MiddlewareFunc(t)
	case HandlerFunc:
		x = t
	case func(Context) error:
		x = HandlerFunc(t)
	default:
		panic("WrapMiddleware's parameter type is incorrect.")
	}
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			if err := x(c); err != nil {
				return err
			}
			return next(c)
		}
	}
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
