package lessgo

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	netContext "golang.org/x/net/context"

	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/websocket"
)

type (
	// Context represents the context of the current HTTP request. It holds request and
	// response objects, path, path parameters, data and registered handler.
	Context interface {
		netContext.Context

		// NetContext returns `http://blog.golang.org/context.Context` interface.
		NetContext() netContext.Context

		// SetNetContext sets `http://blog.golang.org/context.Context` interface.
		SetNetContext(netContext.Context)

		// About Socket's methods.
		Socket() *websocket.Conn
		SetSocket(*websocket.Conn)
		WsRecvJSON(interface{}) error
		WsRecvMsg(*string) error
		WsSendJSON(interface{}) (int, error)
		WsSendMsg(string) (int, error)

		// Request returns `engine.Request` interface.
		Request() *Request

		// Request returns `engine.Response` interface.
		Response() *Response

		// Path returns the registered path for the handler.
		Path() string

		// SetPath sets the registered path for the handler.
		SetPath(string)

		// P returns path parameter by index.
		P(int) string

		// Param returns path parameter by name.
		Param(string) string

		// ParamNames returns path parameter names.
		ParamNames() []string

		// ParamValues returns path parameter values.
		ParamValues() []string

		// SetParam sets path parameter.
		SetParam(name, value string)

		// setParamNames sets path parameter names.
		setParamNames([]string)

		// setParamValues sets path parameter values.
		setParamValues([]string)

		// QueryParam returns the query param for the provided name. It is an alias
		// for `engine.URL#QueryParam()`.
		QueryParam(string) string

		// QueryParams returns the query parameters as map.
		// It is an alias for `engine.URL#QueryParams()`.
		QueryParams() map[string][]string

		// FormValue returns the form field value for the provided name. It is an
		// alias for `engine.Request#FormValue()`.
		FormValue(string) string

		// FormParams returns the form parameters as map.
		// It is an alias for `engine.Request#FormParams()`.
		FormParams() map[string][]string

		// FormFile returns the multipart form file for the provided name. It is an
		// alias for `engine.Request#FormFile()`.
		FormFile(string) (*multipart.FileHeader, error)

		// MultipartForm returns the multipart form.
		// It is an alias for `engine.Request#MultipartForm()`.
		MultipartForm() (*multipart.Form, error)

		// Cookie returns the named cookie provided in the request.
		// It is an alias for `engine.Request#Cookie()`.
		Cookie(string) (*http.Cookie, error)

		// SetCookie adds a `Set-Cookie` header in HTTP response.
		// It is an alias for `engine.Response#SetCookie()`.
		SetCookie(*http.Cookie)

		// Cookies returns the HTTP cookies sent with the request.
		// It is an alias for `engine.Request#Cookies()`.
		Cookies() []*http.Cookie

		// CruSession returns session data store.
		CruSession() session.Store

		// SetSession puts value into session.
		SetSession(name interface{}, value interface{})

		// GetSession gets value from session.
		GetSession(name interface{}) interface{}

		// DelSession removes value from session.
		DelSession(name interface{})

		// SessionRegenerateID regenerates session id for this session.
		// the session data have no changes.
		SessionRegenerateID()

		// DestroySession cleans session data and session cookie.
		DestroySession()

		// Get retrieves data from the context.
		Get(string) interface{}

		// Set saves data in the context.
		Set(string, interface{})

		// Del deletes data from the context.
		Del(string)

		// Exists checks if that key exists in the context.
		Exists(string) bool

		// Bind binds the request body into provided type `i`. The default binder
		// does it based on Content-Type header.
		Bind(interface{}) error

		// Render renders a template with data and sends a text/html response with status
		// code. Templates can be registered using `Echo.SetRenderer()`.
		Render(int, string, interface{}) error

		// HTML sends an HTTP response with status code.
		HTML(int, string) error

		// String sends a string response with status code.
		String(int, string) error

		// JSON sends a JSON response with status code.
		JSON(int, interface{}) error

		// JSONMsg sends a JSON response with common message format.
		JSONMsg(code int, msgcode int, info interface{}) error

		// JSONBlob sends a JSON blob response with status code.
		JSONBlob(int, []byte) error

		// JSONP sends a JSONP response with status code. It uses `callback` to construct
		// the JSONP payload.
		JSONP(int, string, interface{}) error

		// JSONPMsg sends a JSONP response with common message format.
		JSONPMsg(code int, callback string, msgcode int, info interface{}) error

		// XML sends an XML response with status code.
		XML(int, interface{}) error

		// XMLBlob sends a XML blob response with status code.
		XMLBlob(int, []byte) error

		// File sends a response with the content of the file.
		File(string) error

		// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
		// client to save the file.
		Attachment(io.ReadSeeker, string) error

		// NoContent sends a response with no body and a status code.
		NoContent(int) error

		// Redirect redirects the request with status code.
		Redirect(int, string) error

		// Error invokes the registered HTTP error handler. Generally used by middleware.
		Error(err error)

		// Handler returns the matched handler by router.
		Handler() HandlerFunc

		// SetHandler sets the matched handler by router.
		SetHandler(HandlerFunc)

		// Logger returns the `Logger` instance.
		Logger() logs.Logger

		// App returns the `Echo` instance.
		App() *Echo

		// ServeContent sends static content from `io.Reader` and handles caching
		// via `If-Modified-Since` request header. It automatically sets `Content-Type`
		// and `Last-Modified` response headers.
		ServeContent(io.ReadSeeker, string, time.Time) error

		// init context
		init(http.ResponseWriter, *http.Request) error

		// free context
		free()
	}

	context struct {
		netContext netContext.Context
		request    *Request
		response   *Response
		path       string
		pnames     []string
		pvalues    []string
		store      store
		handler    HandlerFunc
		cruSession session.Store
		socket     *websocket.Conn
		echo       *Echo
	}

	store map[string]interface{}

	// Common message format of JSON and JSONP.
	CommMsg struct {
		Code int         `json:"code"`
		Info interface{} `json:"info"`
	}
)

const (
	indexPage = "index.html"
)

var _ Context = new(context)

func (c *context) NetContext() netContext.Context {
	return c.netContext
}

func (c *context) SetNetContext(ctx netContext.Context) {
	c.netContext = ctx
}

func (c *context) Deadline() (deadline time.Time, ok bool) {
	return c.netContext.Deadline()
}

func (c *context) Done() <-chan struct{} {
	return c.netContext.Done()
}

func (c *context) Err() error {
	return c.netContext.Err()
}

func (c *context) Value(key interface{}) interface{} {
	return c.netContext.Value(key)
}

func (c *context) Socket() *websocket.Conn {
	return c.socket
}

func (c *context) SetSocket(conn *websocket.Conn) {
	c.socket = conn
}

func (c *context) WsRecvJSON(v interface{}) error {
	return websocket.JSON.Receive(c.socket, v)
}

func (c *context) WsRecvMsg(v *string) error {
	return websocket.Message.Receive(c.socket, v)
}

func (c *context) WsSendJSON(v interface{}) (int, error) {
	return websocket.JSON.Send(c.socket, v)
}

func (c *context) WsSendMsg(v string) (int, error) {
	return websocket.Message.Send(c.socket, v)
}

func (c *context) Request() *Request {
	return c.request
}

func (c *context) Response() *Response {
	return c.response
}

func (c *context) Path() string {
	return c.path
}

func (c *context) SetPath(p string) {
	c.path = p
}

func (c *context) P(i int) (value string) {
	l := len(c.pnames)
	if i < l {
		value = c.pvalues[i]
	}
	return
}

func (c *context) Param(name string) (value string) {
	l := len(c.pnames)
	for i, n := range c.pnames {
		if n == name && i < l {
			value = c.pvalues[i]
			break
		}
	}
	return
}

func (c *context) ParamNames() []string {
	return c.pnames
}

func (c *context) SetParam(name, value string) {
	l := len(c.pnames)
	for i, n := range c.pnames {
		if n == name && i < l {
			c.pvalues[i] = value
			return
		}
	}
	c.pnames = append(c.pnames, name)
	if len(c.pvalues) > l {
		c.pvalues[l] = value
	} else {
		c.pvalues = append(c.pvalues, value)
	}
}

func (c *context) ParamValues() []string {
	return c.pvalues
}

func (c *context) setParamNames(names []string) {
	c.pnames = names
}

func (c *context) setParamValues(values []string) {
	c.pvalues = values
}

func (c *context) QueryParam(name string) string {
	return c.request.QueryParam(name)
}

func (c *context) QueryParams() map[string][]string {
	return c.request.QueryParams()
}

func (c *context) FormValue(name string) string {
	return c.request.FormValue(name)
}

func (c *context) FormParams() map[string][]string {
	return c.request.FormParams()
}

func (c *context) FormFile(name string) (*multipart.FileHeader, error) {
	_, fh, err := c.request.FormFile(name)
	return fh, err
}

func (c *context) MultipartForm() (*multipart.Form, error) {
	return c.request.MultipartForm()
}

func (c *context) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

func (c *context) SetCookie(cookie *http.Cookie) {
	c.response.SetCookie(cookie)
}

func (c *context) Cookies() []*http.Cookie {
	return c.request.Cookies()
}

// session data info.
func (c *context) CruSession() session.Store {
	return c.cruSession
}

// SetSession puts value into session.
func (c *context) SetSession(name interface{}, value interface{}) {
	if c.cruSession == nil {
		return
	}
	c.cruSession.Set(name, value)
}

// GetSession gets value from session.
func (c *context) GetSession(name interface{}) interface{} {
	if c.cruSession == nil {
		return nil
	}
	return c.cruSession.Get(name)
}

// DelSession removes value from session.
func (c *context) DelSession(name interface{}) {
	if c.cruSession == nil {
		return
	}
	c.cruSession.Delete(name)
}

// SessionRegenerateID regenerates session id for this session.
// the session data have no changes.
func (c *context) SessionRegenerateID() {
	if c.cruSession == nil {
		return
	}
	c.cruSession.SessionRelease(c.Response().Writer())
	c.cruSession = c.echo.sessions.SessionRegenerateID(c.Response().Writer(), c.Request().Request)
}

// DestroySession cleans session data and session cookie.
func (c *context) DestroySession() {
	if c.cruSession == nil {
		return
	}
	c.cruSession.Flush()
	c.cruSession = nil
	c.echo.sessions.SessionDestroy(c.Response().Writer(), c.Request().Request)
}

func (c *context) Set(key string, val interface{}) {
	if c.store == nil {
		c.store = make(store)
	}
	c.store[key] = val
}

func (c *context) Get(key string) interface{} {
	return c.store[key]
}

func (c *context) Del(key string) {
	delete(c.store, key)
}

func (c *context) Exists(key string) bool {
	_, ok := c.store[key]
	return ok
}

func (c *context) Bind(i interface{}) error {
	return c.echo.binder.Bind(i, c)
}

func (c *context) Render(code int, name string, data interface{}) (err error) {
	if c.echo.renderer == nil {
		return ErrRendererNotRegistered
	}
	buf := new(bytes.Buffer)
	if err = c.echo.renderer.Render(buf, name, data, c); err != nil {
		return
	}
	c.response.Header().Set(HeaderContentType, MIMETextHTMLCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	_, err = c.response.Write(buf.Bytes())
	return
}

func (c *context) HTML(code int, html string) (err error) {
	c.response.Header().Set(HeaderContentType, MIMETextHTMLCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	_, err = c.response.Write([]byte(html))
	return
}

func (c *context) String(code int, s string) (err error) {
	c.response.Header().Set(HeaderContentType, MIMETextPlainCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	_, err = c.response.Write([]byte(s))
	return
}

func (c *context) JSON(code int, i interface{}) (err error) {
	b, err := json.Marshal(i)
	if c.echo.Debug() {
		b, err = json.MarshalIndent(i, "", "  ")
	}
	if err != nil {
		return err
	}
	return c.JSONBlob(code, b)
}

func (c *context) JSONMsg(code int, msgcode int, info interface{}) (err error) {
	b, err := json.Marshal(CommMsg{
		Code: msgcode,
		Info: info,
	})
	if c.echo.Debug() {
		b, err = json.MarshalIndent(info, "", "  ")
	}
	if err != nil {
		return err
	}

	return c.JSONBlob(code, b)
}

func (c *context) JSONBlob(code int, b []byte) (err error) {
	c.response.Header().Set(HeaderContentType, MIMEApplicationJSONCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	_, err = c.response.Write(b)
	return
}

func (c *context) JSONP(code int, callback string, i interface{}) (err error) {
	b, err := json.Marshal(i)
	if err != nil {
		return err
	}
	c.response.Header().Set(HeaderContentType, MIMEApplicationJavaScriptCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	if _, err = c.response.Write([]byte(callback + "(")); err != nil {
		return
	}
	if _, err = c.response.Write(b); err != nil {
		return
	}
	_, err = c.response.Write([]byte(");"))
	return
}

func (c *context) JSONPMsg(code int, callback string, msgcode int, info interface{}) (err error) {
	b, err := json.Marshal(CommMsg{
		Code: msgcode,
		Info: info,
	})
	if err != nil {
		return err
	}
	c.response.Header().Set(HeaderContentType, MIMEApplicationJavaScriptCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	if _, err = c.response.Write([]byte(callback + "(")); err != nil {
		return
	}
	if _, err = c.response.Write(b); err != nil {
		return
	}
	_, err = c.response.Write([]byte(");"))
	return
}

func (c *context) XML(code int, i interface{}) (err error) {
	b, err := xml.Marshal(i)
	if c.echo.Debug() {
		b, err = xml.MarshalIndent(i, "", "  ")
	}
	if err != nil {
		return err
	}
	return c.XMLBlob(code, b)
}

func (c *context) XMLBlob(code int, b []byte) (err error) {
	c.response.Header().Set(HeaderContentType, MIMEApplicationXMLCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	if _, err = c.response.Write([]byte(xml.Header)); err != nil {
		return
	}
	_, err = c.response.Write(b)
	return
}

func (c *context) File(file string) error {
	if c.echo.MemoryCacheEnable() {
		f, fi, exist := c.echo.memoryCache.GetCacheFile(file)
		if !exist {
			return ErrNotFound
		}
		return c.ServeContent(f, fi.Name(), fi.ModTime())
	}
	f, err := os.Open(file)
	if err != nil {
		return ErrNotFound
	}
	defer f.Close()

	fi, _ := f.Stat()
	if fi.IsDir() {
		file = filepath.Join(file, indexPage)
		f, err = os.Open(file)
		if err != nil {
			return ErrNotFound
		}
		fi, _ = f.Stat()
	}
	return c.ServeContent(f, fi.Name(), fi.ModTime())
}

func (c *context) Attachment(r io.ReadSeeker, name string) (err error) {
	c.response.Header().Set(HeaderContentType, ContentTypeByExtension(name))
	c.response.Header().Set(HeaderContentDisposition, "attachment; filename="+name)
	c.freeSession()
	c.response.WriteHeader(http.StatusOK)
	_, err = io.Copy(c.response, r)
	return
}

func (c *context) NoContent(code int) error {
	c.freeSession()
	c.response.WriteHeader(code)
	return nil
}

func (c *context) Redirect(code int, url string) error {
	if code < http.StatusMultipleChoices || code > http.StatusTemporaryRedirect {
		return ErrInvalidRedirectCode
	}
	c.response.Header().Set(HeaderLocation, url)
	c.freeSession()
	c.response.WriteHeader(code)
	return nil
}

func (c *context) Error(err error) {
	c.echo.httpErrorHandler(err, c)
}

func (c *context) App() *Echo {
	return c.echo
}

func (c *context) Handler() HandlerFunc {
	return c.handler
}

func (c *context) SetHandler(h HandlerFunc) {
	c.handler = h
}

func (c *context) Logger() logs.Logger {
	return c.echo.logger
}

func (c *context) ServeContent(content io.ReadSeeker, name string, modtime time.Time) error {
	req := c.Request()
	resp := c.Response()

	if t, err := time.Parse(http.TimeFormat, req.Header.Get(HeaderIfModifiedSince)); err == nil && modtime.Before(t.Add(1*time.Second)) {
		resp.Header().Del(HeaderContentType)
		resp.Header().Del(HeaderContentLength)
		return c.NoContent(http.StatusNotModified)
	}

	resp.Header().Set(HeaderContentType, ContentTypeByExtension(name))
	resp.Header().Set(HeaderLastModified, modtime.UTC().Format(http.TimeFormat))
	c.freeSession()
	resp.WriteHeader(http.StatusOK)
	_, err := io.Copy(resp, content)
	return err
}

func (c *context) freeSession() {
	if c.cruSession != nil {
		c.cruSession.SessionRelease(c.Response().Writer())
		c.cruSession = nil
	}
}

func (c *context) init(rw http.ResponseWriter, req *http.Request) (err error) {
	if c.echo.sessions != nil {
		c.cruSession, err = c.echo.sessions.SessionStart(rw, req)
		if err != nil {
			c.NoContent(503)
			return err
		}
	}
	c.response.SetWriter(rw)
	c.request.SetRequest(req)
	return err
}

func (c *context) free() {
	c.handler = notFoundHandler
	c.netContext = nil
	c.socket = nil
	c.store = make(store)
	for i := len(c.pnames) - 1; i >= 0; i-- {
		c.pvalues[i] = ""
	}
	c.pnames = []string{}
	c.response.free()
	c.request.free()
}

// ContentTypeByExtension returns the MIME type associated with the file based on
// its extension. It returns `application/octet-stream` incase MIME type is not
// found.
func ContentTypeByExtension(name string) (t string) {
	if t = mime.TypeByExtension(filepath.Ext(name)); t == "" {
		t = MIMEOctetStream
	}
	return
}
