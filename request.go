package lessgo

import (
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"strings"
)

type Request struct {
	*http.Request
	query          url.Values
	realRemoteAddr string
}

// 文件上传默认内存缓存大小，默认值是 1 << 32 (32MB)。
var (
	MaxMemory int64 = 32 << 20
)

func (r *Request) IsTLS() bool {
	return r.Request.TLS != nil
}

func (r *Request) Scheme() string {
	if r.IsTLS() {
		return "https"
	}
	return "http"
}

func (r *Request) RealRemoteAddr() string {
	if len(r.realRemoteAddr) == 0 {
		r.realRemoteAddr = r.RemoteAddr
		if ip := r.Header.Get(HeaderXRealIP); ip != "" {
			r.realRemoteAddr = ip
		} else if ip = r.Header.Get(HeaderXForwardedFor); ip != "" {
			r.realRemoteAddr = ip
		} else {
			r.realRemoteAddr, _, _ = net.SplitHostPort(r.realRemoteAddr)
		}
	}
	return r.realRemoteAddr
}

func (r *Request) ContentLength() int {
	return int(r.Request.ContentLength)
}

func (r *Request) SetBody(reader io.Reader) {
	r.Request.Body = ioutil.NopCloser(reader)
}

func (r *Request) QueryParam(name string) string {
	if r.query == nil {
		r.query = r.URL.Query()
	}
	return r.query.Get(name)
}

func (r *Request) QueryParams() map[string][]string {
	if r.query == nil {
		r.query = r.URL.Query()
	}
	return map[string][]string(r.query)
}

func (r *Request) FormParams() map[string][]string {
	if strings.HasPrefix(r.Header.Get(HeaderContentType), MIMEMultipartForm) {
		if err := r.ParseMultipartForm(MaxMemory); err != nil {
			Logger().Error("%v", err)
		}
	} else {
		if err := r.ParseForm(); err != nil {
			Logger().Error("%v", err)
		}
	}
	return map[string][]string(r.Request.Form)
}

func (r *Request) MultipartForm() (*multipart.Form, error) {
	err := r.ParseMultipartForm(MaxMemory)
	return r.Request.MultipartForm, err
}

func (r *Request) reset(req *http.Request) {
	r.Request = req
	r.query = nil
	r.realRemoteAddr = ""
}
