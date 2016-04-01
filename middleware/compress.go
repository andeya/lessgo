package middleware

import (
	"compress/gzip"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"

	"github.com/lessgo/lessgo"
	"github.com/lessgo/lessgo/engine"
)

type (
	// GzipConfig defines the config for gzip middleware.
	GzipConfig struct {
		// Level is the gzip level.
		// Optional with default value as `DefaultGzipConfig.Level`.
		Level int
	}

	gzipResponseWriter struct {
		engine.Response
		io.Writer
	}
)

var (
	// DefaultGzipConfig is the default gzip middleware config.
	DefaultGzipConfig = GzipConfig{
		Level: -1,
	}
)

// Gzip returns a middleware which compresses HTTP response using gzip compression
// scheme.
func Gzip() lessgo.MiddlewareFunc {
	return GzipFromConfig(DefaultGzipConfig)
}

// GzipFromConfig return gzip middleware from config.
// See `Gzip()`.
func GzipFromConfig(config GzipConfig) lessgo.MiddlewareFunc {
	// Defaults
	if config.Level == 0 {
		config.Level = DefaultGzipConfig.Level
	}
	pool := gzipPool(config)
	scheme := "gzip"

	return func(next lessgo.Handler) lessgo.Handler {
		return lessgo.HandlerFunc(func(c lessgo.Context) error {
			c.Response().Header().Add(lessgo.Vary, lessgo.AcceptEncoding)
			if strings.Contains(c.Request().Header().Get(lessgo.AcceptEncoding), scheme) {
				rw := c.Response().Writer()
				gw := pool.Get().(*gzip.Writer)
				gw.Reset(rw)
				defer func() {
					if c.Response().Size() == 0 {
						// We have to reset response to it's pristine state when
						// nothing is written to body or error is returned.
						// See issue #424, #407.
						c.Response().SetWriter(rw)
						c.Response().Header().Del(lessgo.ContentEncoding)
						gw.Reset(ioutil.Discard)
					}
					gw.Close()
					pool.Put(gw)
				}()
				g := gzipResponseWriter{Response: c.Response(), Writer: gw}
				c.Response().Header().Set(lessgo.ContentEncoding, scheme)
				c.Response().SetWriter(g)
			}
			return next.Handle(c)
		})
	}
}

func (g gzipResponseWriter) Write(b []byte) (int, error) {
	if g.Header().Get(lessgo.ContentType) == "" {
		g.Header().Set(lessgo.ContentType, http.DetectContentType(b))
	}
	return g.Writer.Write(b)
}

func gzipPool(config GzipConfig) sync.Pool {
	return sync.Pool{
		New: func() interface{} {
			w, _ := gzip.NewWriterLevel(ioutil.Discard, config.Level)
			return w
		},
	}
}
