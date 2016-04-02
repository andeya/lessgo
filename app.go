/*
Package echo implements a fast and unfancy micro web framework for Go.

Learn more at https://labstack.com/echo
*/
package lessgo

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
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
)

type (
	// Echo is the top-level framework instance.
	Echo struct {
		prefix           string
		middleware       []Middleware
		head             Handler
		pristineHead     Handler
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

	// Middleware defines an interface for middleware via `Handle(Handler) Handler`
	// function.
	Middleware interface {
		Handle(Handler) Handler
	}

	// MiddlewareFunc is an adapter to allow the use of `func(Handler) Handler` as
	// middleware.
	MiddlewareFunc func(Handler) Handler

	// Handler defines an interface to server HTTP requests via `Handle(Context)`
	// function.
	Handler interface {
		Handle(Context) error
	}

	// HandlerFunc is an adapter to allow the use of `func(Context)` as an HTTP
	// handler.
	HandlerFunc func(Context) error

	// HTTPErrorHandler is a centralized HTTP error handler.
	HTTPErrorHandler func(error, Context)

	// Binder is the interface that wraps the Bind function.
	Binder interface {
		Bind(interface{}, Context) error
	}

	binder struct {
	}

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

// Media types
const (
	ApplicationJSON                  = "application/json"
	ApplicationJSONCharsetUTF8       = ApplicationJSON + "; " + CharsetUTF8
	ApplicationJavaScript            = "application/javascript"
	ApplicationJavaScriptCharsetUTF8 = ApplicationJavaScript + "; " + CharsetUTF8
	ApplicationXML                   = "application/xml"
	ApplicationXMLCharsetUTF8        = ApplicationXML + "; " + CharsetUTF8
	ApplicationForm                  = "application/x-www-form-urlencoded"
	ApplicationProtobuf              = "application/protobuf"
	ApplicationMsgpack               = "application/msgpack"
	TextHTML                         = "text/html"
	TextHTMLCharsetUTF8              = TextHTML + "; " + CharsetUTF8
	TextPlain                        = "text/plain"
	TextPlainCharsetUTF8             = TextPlain + "; " + CharsetUTF8
	MultipartForm                    = "multipart/form-data"
	OctetStream                      = "application/octet-stream"
)

// Charset
const (
	CharsetUTF8 = "charset=utf-8"
)

// Headers
const (
	AcceptEncoding     = "Accept-Encoding"
	Authorization      = "Authorization"
	ContentDisposition = "Content-Disposition"
	ContentEncoding    = "Content-Encoding"
	ContentLength      = "Content-Length"
	ContentType        = "Content-Type"
	IfModifiedSince    = "If-Modified-Since"
	LastModified       = "Last-Modified"
	Location           = "Location"
	Upgrade            = "Upgrade"
	Vary               = "Vary"
	WWWAuthenticate    = "WWW-Authenticate"
	XForwardedFor      = "X-Forwarded-For"
	XRealIP            = "X-Real-IP"
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
	ErrUnsupportedMediaType  = NewHTTPError(http.StatusUnsupportedMediaType)
	ErrNotFound              = NewHTTPError(http.StatusNotFound)
	ErrUnauthorized          = NewHTTPError(http.StatusUnauthorized)
	ErrMethodNotAllowed      = NewHTTPError(http.StatusMethodNotAllowed)
	ErrRendererNotRegistered = errors.New("renderer not registered")
	ErrInvalidRedirectCode   = errors.New("invalid redirect status code")
)

// Error handlers
var (
	notFoundHandler = HandlerFunc(func(c Context) error {
		return ErrNotFound
	})

	methodNotAllowedHandler = HandlerFunc(func(c Context) error {
		return ErrMethodNotAllowed
	})
)

// New creates an instance of Echo.
func New() (e *Echo) {
	e = &Echo{maxParam: new(int)}
	e.pool.New = func() interface{} {
		return NewContext(nil, nil, e)
	}
	e.router = NewRouter(e)
	e.middleware = []Middleware{e.router}
	e.head = HandlerFunc(func(c Context) error {
		fmt.Println("-------------------head(目测根本不会调用到)-----------------")
		return c.Handle(c)
	})
	e.pristineHead = e.head
	e.chainMiddleware()

	// Defaults
	e.SetHTTPErrorHandler(e.DefaultHTTPErrorHandler)
	e.SetBinder(&binder{})
	e.logger = logs.GlobalLogger
	e.logger.AddAdapter("console", "")
	e.logger.AddAdapter("file", `{"filename":"Logger/lessgo.log"}`)
	e.SetDebug(true)
	return
}

// Handle chains middleware.
func (f MiddlewareFunc) Handle(h Handler) Handler {
	return f(h)
}

// Handle serves HTTP request.
func (f HandlerFunc) Handle(c Context) error {
	return f(c)
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

// PreUse adds middlewares to the beginning of chain.
func (e *Echo) PreUse(middleware ...Middleware) {
	e.routerIndex += len(middleware)
	e.middleware = append(middleware, e.middleware...)
	e.chainMiddleware()
}

// SufUse adds middlewares to the end of chain.
func (e *Echo) SufUse(middleware ...Middleware) {
	e.middleware = append(e.middleware, middleware...)
	e.chainMiddleware()
}

// BeforeUse adds middlewares to the chain which is run before router.
func (e *Echo) BeforeUse(middleware ...Middleware) {
	chain := make([]Middleware, e.routerIndex)
	copy(chain, e.middleware[:e.routerIndex])
	chain = append(chain, middleware...)
	e.middleware = append(chain, e.middleware[e.routerIndex:]...)
	e.routerIndex += len(middleware)
	e.chainMiddleware()
}

// AfterUse adds middlewares to the chain which is run after router.
func (e *Echo) AfterUse(middleware ...Middleware) {
	chain := make([]Middleware, e.routerIndex+1)
	copy(chain, e.middleware[:e.routerIndex+1])
	chain = append(chain, middleware...)
	e.middleware = append(chain, e.middleware[e.routerIndex+1:]...)
	e.chainMiddleware()
}

func (e *Echo) chainMiddleware() {
	e.head = e.pristineHead
	for i := len(e.middleware) - 1; i >= 0; i-- {
		e.head = e.middleware[i].Handle(e.head)
	}
}

// Connect registers a new CONNECT route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Connect(path string, h Handler, m ...Middleware) {
	e.add(CONNECT, path, h, m...)
}

// Delete registers a new DELETE route for a path with matching handler in the router
// with optional route-level middleware.
func (e *Echo) Delete(path string, h Handler, m ...Middleware) {
	e.add(DELETE, path, h, m...)
}

// Get registers a new GET route for a path with matching handler in the router
// with optional route-level middleware.
func (e *Echo) Get(path string, h Handler, m ...Middleware) {
	e.add(GET, path, h, m...)
}

// Head registers a new HEAD route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Head(path string, h Handler, m ...Middleware) {
	e.add(HEAD, path, h, m...)
}

// Options registers a new OPTIONS route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Options(path string, h Handler, m ...Middleware) {
	e.add(OPTIONS, path, h, m...)
}

// Patch registers a new PATCH route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Patch(path string, h Handler, m ...Middleware) {
	e.add(PATCH, path, h, m...)
}

// Post registers a new POST route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Post(path string, h Handler, m ...Middleware) {
	e.add(POST, path, h, m...)
}

// Put registers a new PUT route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Put(path string, h Handler, m ...Middleware) {
	e.add(PUT, path, h, m...)
}

// Trace registers a new TRACE route for a path with matching handler in the
// router with optional route-level middleware.
func (e *Echo) Trace(path string, h Handler, m ...Middleware) {
	e.add(TRACE, path, h, m...)
}

// Any registers a new route for all HTTP methods and path with matching handler
// in the router with optional route-level middleware.
func (e *Echo) Any(path string, handler Handler, middleware ...Middleware) {
	for _, m := range methods {
		e.add(m, path, handler, middleware...)
	}
}

// Match registers a new route for multiple HTTP methods and path with matching
// handler in the router with optional route-level middleware.
func (e *Echo) Match(methods []string, path string, handler Handler, middleware ...Middleware) {
	for _, m := range methods {
		e.add(m, path, handler, middleware...)
	}
}

// Static serves files from provided `root` directory for `/<prefix>*` HTTP path.
func (e *Echo) Static(prefix, root string) {
	e.Get(prefix+"*", HandlerFunc(func(c Context) error {
		return c.File(path.Join(root, c.P(0))) // Param `_`
	}))
}

// File serves provided file for `/<path>` HTTP path.
func (e *Echo) File(path, file string) {
	e.Get(path, HandlerFunc(func(c Context) error {
		return c.File(file)
	}))
}

func (e *Echo) add(method, path string, handler Handler, middleware ...Middleware) {
	name := handlerName(handler)
	e.router.Add(method, path, HandlerFunc(func(c Context) error {
		h := handler
		// Chain middleware
		for i := len(middleware) - 1; i >= 0; i-- {
			h = middleware[i].Handle(h)
		}
		return h.Handle(c)
	}), e)
	r := Route{
		Method:  method,
		Path:    path,
		Handler: name,
	}
	e.router.routes = append(e.router.routes, r)
	e.logger.Sys("| %-7s | %-30s | %v", method, path, name)
}

// Group creates a new router group with prefix and optional group-level middleware.
func (e *Echo) Group(prefix string, m ...Middleware) (g *Group) {
	g = &Group{prefix: prefix, echo: e}
	g.Use(m...)
	// Dummy handler so group can be used with static middleware.
	g.Get("", HandlerFunc(func(c Context) error {
		return c.NoContent(http.StatusNotFound)
	}))
	return
}

// URI generates a URI from handler.
func (e *Echo) URI(handler Handler, params ...interface{}) string {
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
func (e *Echo) URL(h Handler, params ...interface{}) string {
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

	c := e.pool.Get().(*context)
	c.Reset(rq, rs)

	// Execute chain
	if err := e.head.Handle(c); err != nil {
		e.httpErrorHandler(err, c)
	}

	e.pool.Put(c)
}

// Run starts the HTTP server.
func (e *Echo) Run(s engine.Server) {
	s.SetHandler(e)
	s.SetLogger(e.logger)
	e.logger.Error("%v", s.Start())
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

func (binder) Bind(i interface{}, c Context) (err error) {
	rq := c.Request()
	ct := rq.Header().Get(ContentType)
	err = ErrUnsupportedMediaType
	if strings.HasPrefix(ct, ApplicationJSON) {
		if err = json.NewDecoder(rq.Body()).Decode(i); err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
	} else if strings.HasPrefix(ct, ApplicationXML) {
		if err = xml.NewDecoder(rq.Body()).Decode(i); err != nil {
			err = NewHTTPError(http.StatusBadRequest, err.Error())
		}
	}
	return
}

// WrapMiddleware wrap `echo.Handler` into `echo.MiddlewareFunc`.
func WrapMiddleware(h interface{}) MiddlewareFunc {
	var x Handler
	switch t := h.(type) {
	case MiddlewareFunc:
		return t
	case Handler:
		x = t
	case HandlerFunc:
		x = Handler(t)
	case func(Context) error:
		x = Handler(HandlerFunc(t))
	default:
		panic("WrapMiddleware's parameter type is incorrect.")
	}
	return func(next Handler) Handler {
		return HandlerFunc(func(c Context) error {
			if err := x.Handle(c); err != nil {
				return err
			}
			return next.Handle(c)
		})
	}
}

func handlerName(h Handler) string {
	t := reflect.ValueOf(h).Type()
	if t.Kind() == reflect.Func {
		return runtime.FuncForPC(reflect.ValueOf(h).Pointer()).Name()
	}
	return t.String()
}
