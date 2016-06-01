package lessgo

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io"
	"io/ioutil"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lessgo/lessgo/logs"
	"github.com/lessgo/lessgo/session"
	"github.com/lessgo/lessgo/utils"
	"github.com/lessgo/lessgo/websocket"
)

type (
	Context struct {
		request        *http.Request
		response       *Response
		path           string
		realRemoteAddr string
		query          url.Values
		pnames         []string
		pvalues        []string
		store          store
		cruSession     session.Store
		socket         *websocket.Conn
	}

	store map[string]interface{}

	// Common message format of JSON and JSONP.
	CommMsg struct {
		Code int         `json:"code"`
		Info interface{} `json:"info,omitempty"`
	}
)

var (
	// 默认页面文件
	indexPage = "index.html"
	// 文件上传默认内存缓存大小，默认值是 1 << 32 (32MB)。
	MaxMemory int64 = 32 << 20
)

func (c *Context) RealRemoteAddr() string {
	if len(c.realRemoteAddr) == 0 {
		c.realRemoteAddr = c.request.RemoteAddr
		if ip := c.request.Header.Get(HeaderXRealIP); ip != "" {
			c.realRemoteAddr = ip
		} else if ip = c.request.Header.Get(HeaderXForwardedFor); ip != "" {
			c.realRemoteAddr = ip
		} else {
			c.realRemoteAddr, _, _ = net.SplitHostPort(c.realRemoteAddr)
		}
	}
	return c.realRemoteAddr
}

func (c *Context) Socket() *websocket.Conn {
	return c.socket
}

func (c *Context) SetSocket(conn *websocket.Conn) {
	c.socket = conn
}

func (c *Context) WsRecvJSON(v interface{}) error {
	return websocket.JSON.Receive(c.socket, v)
}

func (c *Context) WsRecvMsg(v *string) error {
	return websocket.Message.Receive(c.socket, v)
}

func (c *Context) WsSendJSON(v interface{}) (int, error) {
	return websocket.JSON.Send(c.socket, v)
}

func (c *Context) WsSendMsg(v string) (int, error) {
	return websocket.Message.Send(c.socket, v)
}

func (c *Context) Request() *http.Request {
	return c.request
}

func (c *Context) SetRequestBody(reader io.Reader) {
	c.request.Body = ioutil.NopCloser(reader)
}

func (c *Context) Response() *Response {
	return c.response
}

func (c *Context) IsTLS() bool {
	return c.request.TLS != nil
}

func (c *Context) Scheme() string {
	if c.IsTLS() {
		return "https"
	}
	return "http"
}

// Path returns the registered path for the handler.
func (c *Context) Path() string {
	return c.path
}

// SetPath sets the registered path for the handler.
func (c *Context) SetPath(p string) {
	c.path = p
}

// P returns path parameter by index.
func (c *Context) P(i int) (value string) {
	l := len(c.pnames)
	if i < l {
		value = c.pvalues[i]
	}
	return
}

// Param returns path parameter by name.
func (c *Context) Param(name string) (value string) {
	l := len(c.pnames)
	for i, n := range c.pnames {
		if n == name && i < l {
			value = c.pvalues[i]
			break
		}
	}
	return
}

// ParamNames returns path parameter names.
func (c *Context) ParamNames() []string {
	return c.pnames
}

// SetParam sets path parameter.
func (c *Context) SetParam(name, value string) {
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

// ParamValues returns path parameter values.
func (c *Context) ParamValues() []string {
	return c.pvalues
}

// QueryParam returns the query param for the provided name.
func (c *Context) QueryParam(name string) string {
	if c.query == nil {
		c.query = c.request.URL.Query()
	}
	return c.query.Get(name)
}

// QueryParams returns the query parameters.
func (c *Context) QueryParams() url.Values {
	if c.query == nil {
		c.query = c.request.URL.Query()
	}
	return c.query
}

// FormValue returns the form field value for the provided name.
func (c *Context) FormValue(name string) string {
	return c.request.FormValue(name)
}

// FormParams returns the form parameters as map.
func (c *Context) FormParams() url.Values {
	if strings.HasPrefix(c.request.Header.Get(HeaderContentType), MIMEMultipartForm) {
		if err := c.request.ParseMultipartForm(MaxMemory); err != nil {
			Log.Error("%v", err)
		}
	} else {
		if err := c.request.ParseForm(); err != nil {
			Log.Error("%v", err)
		}
	}
	return c.request.Form
}

// FormFile returns the multipart form file for the provided name.
func (c *Context) FormFile(name string) (*multipart.FileHeader, error) {
	_, fh, err := c.request.FormFile(name)
	return fh, err
}

// MultipartForm returns the multipart form.
func (c *Context) MultipartForm() (*multipart.Form, error) {
	err := c.request.ParseMultipartForm(MaxMemory)
	return c.request.MultipartForm, err
}

// SaveFile saves the file *Context.FormFile to UPLOADS_DIR,
// character "?" indicates that the original file name.
// for example newfname="a/?" -> UPLOADS_DIR/a/fname.
func (c *Context) SaveFile(pname string, cover bool, newfname ...string) (fullname string, size int64, err error) {
	f, fh, err := c.request.FormFile(pname)
	if err != nil {
		return
	}
	defer func() {
		err2 := f.Close()
		if err2 != nil && err == nil {
			err = err2
		}
	}()
	if len(newfname) > 0 {
		fullname = filepath.Join(UPLOADS_DIR, strings.Replace(newfname[0], "?", fh.Filename, -1))
		p, _ := filepath.Split(fullname)
		err = os.MkdirAll(p, 0777)
		if err != nil {
			return
		}
	} else {
		fullname = filepath.Join(UPLOADS_DIR, fh.Filename)
	}
	if utils.FileExists(fullname) && !cover {
		idx := strings.LastIndex(fullname, filepath.Ext(fullname))
		fullname = fullname[:idx] + "(2)" + fullname[idx:]
	}
	f2, err := os.OpenFile(fullname, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	size, err = io.Copy(f2, f)
	defer func() {
		err3 := f2.Close()
		if err3 != nil && err == nil {
			err = err3
		}
	}()
	return
}

// Cookie returns the named cookie provided in the request.
func (c *Context) Cookie(name string) (*http.Cookie, error) {
	return c.request.Cookie(name)
}

// SetCookie adds a `Set-Cookie` header in HTTP response.
func (c *Context) SetCookie(cookie *http.Cookie) {
	c.response.SetCookie(cookie)
}

// Cookies returns the HTTP cookies sent with the request.
func (c *Context) Cookies() []*http.Cookie {
	return c.request.Cookies()
}

// CruSession returns session data info.
func (c *Context) CruSession() session.Store {
	return c.cruSession
}

// SetSession puts value into session.
func (c *Context) SetSession(name interface{}, value interface{}) {
	if c.cruSession == nil {
		return
	}
	c.cruSession.Set(name, value)
}

// GetSession gets value from session.
func (c *Context) GetSession(name interface{}) interface{} {
	if c.cruSession == nil {
		return nil
	}
	return c.cruSession.Get(name)
}

// DelSession removes value from session.
func (c *Context) DelSession(name interface{}) {
	if c.cruSession == nil {
		return
	}
	c.cruSession.Delete(name)
}

// SessionRegenerateID regenerates session id for this session.
// the session data have no changes.
func (c *Context) SessionRegenerateID() {
	if c.cruSession == nil {
		return
	}
	c.cruSession.SessionRelease(c.response.Writer())
	c.cruSession = app.sessions.SessionRegenerateID(c.response.Writer(), c.request)
}

// DestroySession cleans session data and session cookie.
func (c *Context) DestroySession() {
	if c.cruSession == nil {
		return
	}
	c.cruSession.Flush()
	c.cruSession = nil
	app.sessions.SessionDestroy(c.response.Writer(), c.request)
}

// Get retrieves data from the context.
func (c *Context) Set(key string, val interface{}) {
	if c.store == nil {
		c.store = make(store)
	}
	c.store[key] = val
}

// Set saves data in the context.
func (c *Context) Get(key string) interface{} {
	return c.store[key]
}

// Del deletes data from the context.
func (c *Context) Del(key string) {
	delete(c.store, key)
}

// Contains checks if the key exists in the context.
func (c *Context) Contains(key string) bool {
	_, ok := c.store[key]
	return ok
}

// Bind binds the request body into provided type `i`. The default binder
// does it based on Content-Type header.
func (c *Context) Bind(i interface{}) error {
	return app.binder.Bind(i, c)
}

// Render renders a template with data and sends a text/html response with status
// code. Templates can be registered using `App.SetRenderer()`.
func (c *Context) Render(code int, name string, data interface{}) (err error) {
	if app.renderer == nil {
		return ErrRendererNotRegistered
	}
	buf := new(bytes.Buffer)
	if err = app.renderer.Render(buf, name, data, c); err != nil {
		return
	}
	c.response.Header().Set(HeaderContentType, MIMETextHTMLCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	_, err = c.response.Write(buf.Bytes())
	return
}

// HTML sends an HTTP response with status code.
func (c *Context) HTML(code int, html string) (err error) {
	c.response.Header().Set(HeaderContentType, MIMETextHTMLCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	_, err = c.response.Write(utils.String2Bytes(html))
	return
}

// String sends a string response with status code.
func (c *Context) String(code int, s string) (err error) {
	c.response.Header().Set(HeaderContentType, MIMETextPlainCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	_, err = c.response.Write(utils.String2Bytes(s))
	return
}

// JSON sends a JSON response with status code.
func (c *Context) JSON(code int, i interface{}) (err error) {
	var b []byte
	if Debug() {
		b, err = json.MarshalIndent(i, "", "  ")
	} else {
		b, err = json.Marshal(i)
	}
	if err != nil {
		return err
	}
	return c.JSONBlob(code, b)
}

// JSON with default format.
func (c *Context) JSONMsg(code int, msgcode int, info interface{}) (err error) {
	var b []byte
	i := CommMsg{
		Code: msgcode,
		Info: info,
	}
	if Debug() {
		b, err = json.MarshalIndent(i, "", "  ")
	} else {
		b, err = json.Marshal(i)
	}
	if err != nil {
		return err
	}

	return c.JSONBlob(code, b)
}

// JSONBlob sends a JSON blob response with status code.
func (c *Context) JSONBlob(code int, b []byte) (err error) {
	c.response.Header().Set(HeaderContentType, MIMEApplicationJSONCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	_, err = c.response.Write(b)
	return
}

// JSONP sends a JSONP response with status code. It uses `callback` to construct
// the JSONP payload.
func (c *Context) JSONP(code int, callback string, i interface{}) (err error) {
	var b []byte
	if Debug() {
		b, err = json.MarshalIndent(i, "", "  ")
	} else {
		b, err = json.Marshal(i)
	}
	if err != nil {
		return err
	}
	c.response.Header().Set(HeaderContentType, MIMEApplicationJavaScriptCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	if _, err = c.response.Write(utils.String2Bytes(callback + "(")); err != nil {
		return
	}
	if _, err = c.response.Write(b); err != nil {
		return
	}
	_, err = c.response.Write(utils.String2Bytes(");"))
	return
}

// JSONP with default format.
func (c *Context) JSONPMsg(code int, callback string, msgcode int, info interface{}) (err error) {
	var b []byte
	i := CommMsg{
		Code: msgcode,
		Info: info,
	}
	if Debug() {
		b, err = json.MarshalIndent(i, "", "  ")
	} else {
		b, err = json.Marshal(i)
	}
	if err != nil {
		return err
	}
	c.response.Header().Set(HeaderContentType, MIMEApplicationJavaScriptCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	if _, err = c.response.Write(utils.String2Bytes(callback + "(")); err != nil {
		return
	}
	if _, err = c.response.Write(b); err != nil {
		return
	}
	_, err = c.response.Write(utils.String2Bytes(");"))
	return
}

// XML sends an XML response with status code.
func (c *Context) XML(code int, i interface{}) (err error) {
	b, err := xml.Marshal(i)
	if Debug() {
		b, err = xml.MarshalIndent(i, "", "  ")
	}
	if err != nil {
		return err
	}
	return c.XMLBlob(code, b)
}

// XMLBlob sends a XML blob response with status code.
func (c *Context) XMLBlob(code int, b []byte) (err error) {
	c.response.Header().Set(HeaderContentType, MIMEApplicationXMLCharsetUTF8)
	c.freeSession()
	c.response.WriteHeader(code)
	if _, err = c.response.Write(utils.String2Bytes(xml.Header)); err != nil {
		return
	}
	_, err = c.response.Write(b)
	return
}

// File sends a response with the content of the file.
func (c *Context) File(file string) error {
	if app.MemoryCacheEnable() {
		f, fi, exist := app.memoryCache.GetCacheFile(file)
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

// Attachment sends a response from `io.ReaderSeeker` as attachment, prompting
// client to save the file.
func (c *Context) Attachment(r io.ReadSeeker, name string) (err error) {
	c.response.Header().Set(HeaderContentType, ContentTypeByExtension(name))
	c.response.Header().Set(HeaderContentDisposition, "attachment; filename="+name)
	c.freeSession()
	c.response.WriteHeader(http.StatusOK)
	_, err = io.Copy(c.response, r)
	return
}

// NoContent sends a response with no body and a status code.
func (c *Context) NoContent(code int) error {
	c.freeSession()
	c.response.WriteHeader(code)
	return nil
}

// Redirect redirects the request with status code.
func (c *Context) Redirect(code int, url string) error {
	if code < http.StatusMultipleChoices || code > http.StatusTemporaryRedirect {
		return ErrInvalidRedirectCode
	}
	c.response.Header().Set(HeaderLocation, url)
	c.freeSession()
	c.response.WriteHeader(code)
	return nil
}

// Error invokes the registered HTTP error handler. Generally used by middleware.
func (c *Context) Error(err error) {
	app.router.ErrorPanicHandler(c, err, nil)
}

// Log returns the `Logger` instance.
func (c *Context) Log() logs.Logger {
	return Log
}

// ServeContent sends static content from `io.Reader` and handles caching
// via `If-Modified-Since` request header. It automatically sets `Content-Type`
// and `Last-Modified` response headers.
func (c *Context) ServeContent(content io.ReadSeeker, name string, modtime time.Time) error {
	req := c.request
	resp := c.response

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

func (c *Context) freeSession() {
	if c.cruSession != nil {
		c.cruSession.SessionRelease(c.response.Writer())
		c.cruSession = nil
	}
}

func (c *Context) init(rw http.ResponseWriter, req *http.Request) (err error) {
	c.pnames = c.pnames[:0]
	c.pvalues = c.pvalues[:0]
	if app.sessions != nil {
		c.cruSession, err = app.sessions.SessionStart(rw, req)
		if err != nil {
			c.NoContent(503)
			return err
		}
	}
	c.response.SetWriter(rw)
	c.request = req
	c.store = make(store)
	return err
}

func (c *Context) free() {
	c.freeSession()
	c.socket = nil
	c.store = nil
	c.realRemoteAddr = ""
	c.query = nil
	c.response.free()
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
