package standard

import (
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/lessgo/lessgo"
	"github.com/lessgo/lessgo/engine"
	"github.com/lessgo/lessgo/engine/standard/grace"
	"github.com/lessgo/lessgo/logs"
)

type (
	// Server implements `engine.Server`.
	Server struct {
		*http.Server
		config  engine.Config
		handler engine.Handler
		logger  logs.Logger
		pool    *pool
	}

	pool struct {
		request         sync.Pool
		response        sync.Pool
		responseAdapter sync.Pool
		header          sync.Pool
		url             sync.Pool
	}
)

// New returns `Server` instance with provided listen address.
func New(addr string) engine.Server {
	c := engine.Config{Address: addr}
	return WithConfig(c)
}

// WithTLS returns `Server` instance with provided TLS config.
func WithTLS(addr, certFile, keyFile string) engine.Server {
	c := engine.Config{
		Address:     addr,
		TLSCertfile: certFile,
		TLSKeyfile:  keyFile,
	}
	return WithConfig(c)
}

// WithConfig returns `Server` instance with provided config.
func WithConfig(c engine.Config) engine.Server {
	var s *Server
	s = &Server{
		Server: new(http.Server),
		config: c,
		pool: &pool{
			request: sync.Pool{
				New: func() interface{} {
					return &Request{logger: s.logger}
				},
			},
			response: sync.Pool{
				New: func() interface{} {
					return &Response{logger: s.logger}
				},
			},
			responseAdapter: sync.Pool{
				New: func() interface{} {
					return &responseAdapter{}
				},
			},
			header: sync.Pool{
				New: func() interface{} {
					return &Header{}
				},
			},
			url: sync.Pool{
				New: func() interface{} {
					return &URL{}
				},
			},
		},
		handler: engine.HandlerFunc(func(req engine.Request, res engine.Response) {
			s.logger.Error("handler not set, use `SetHandler()` to set it.")
		}),
		logger: logs.Global,
	}
	s.Server.Addr = c.Address
	s.Server.Handler = s
	s.Server.ReadTimeout = c.ReadTimeout
	s.Server.WriteTimeout = c.WriteTimeout
	return s
}

// SetHandler implements `engine.Server#SetHandler` function.
func (s *Server) SetHandler(h engine.Handler) {
	s.handler = h
}

// SetLogger implements `engine.Server#SetLogger` function.
func (s *Server) SetLogger(l logs.Logger) {
	s.logger = l
}

// Start implements `engine.Server#Start` function.
func (s *Server) Start() (err error) {
	if !s.config.Graceful {
		if s.config.Listener == nil {
			return s.startDefaultListener()
		}
		return s.startCustomListener()
	}

	endRunning := make(chan bool, 1)
	c := s.config
	server := grace.NewServer(c.Address, s.Server, s.logger)
	if c.TLSCertfile != "" && c.TLSKeyfile != "" {
		go func() {
			time.Sleep(20 * time.Microsecond)
			if err = server.ListenAndServeTLS(c.TLSCertfile, c.TLSKeyfile); err != nil {
				err = fmt.Errorf("ListenAndServeTLS: %v, %d", err, os.Getpid())
				time.Sleep(100 * time.Microsecond)
				endRunning <- true
			}
		}()
	} else {
		go func() {
			// server.Network = "tcp4"
			if err = server.ListenAndServe(); err != nil {
				err = fmt.Errorf("ListenAndServe: %v, %d", err, os.Getpid())
				time.Sleep(100 * time.Microsecond)
				endRunning <- true
			}
		}()
	}
	<-endRunning
	return
}

func (s *Server) startDefaultListener() error {
	c := s.config
	if c.TLSCertfile != "" && c.TLSKeyfile != "" {
		return s.ListenAndServeTLS(c.TLSCertfile, c.TLSKeyfile)
	}
	return s.ListenAndServe()
}

func (s *Server) startCustomListener() error {
	return s.Serve(s.config.Listener)
}

// ServeHTTP implements `http.Handler` interface.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Request
	req := s.pool.request.Get().(*Request)
	reqHdr := s.pool.header.Get().(*Header)
	reqURL := s.pool.url.Get().(*URL)
	reqHdr.reset(r.Header)
	reqURL.reset(r.URL)
	req.reset(r, reqHdr, reqURL)

	// Response
	res := s.pool.response.Get().(*Response)
	resAdpt := s.pool.responseAdapter.Get().(*responseAdapter)
	resAdpt.reset(w, res)
	resHdr := s.pool.header.Get().(*Header)
	resHdr.reset(w.Header())
	res.reset(w, resAdpt, resHdr)

	s.handler.ServeHTTP(req, res)

	// Return to pool
	s.pool.request.Put(req)
	s.pool.header.Put(reqHdr)
	s.pool.url.Put(reqURL)
	s.pool.response.Put(res)
	s.pool.header.Put(resHdr)
}

// WrapHandler wraps `http.Handler` into `lessgo.HandlerFunc`.
func WrapHandler(h http.Handler) lessgo.HandlerFunc {
	return func(c lessgo.Context) error {
		req := c.Request().(*Request)
		res := c.Response().(*Response)
		h.ServeHTTP(res.ResponseWriter, req.Request)
		return nil
	}
}

// WrapMiddleware wraps `func(http.Handler) http.Handler` into `lessgo.MiddlewareFunc`
func WrapMiddleware(m func(http.Handler) http.Handler) lessgo.MiddlewareFunc {
	return func(next lessgo.HandlerFunc) lessgo.HandlerFunc {
		return func(c lessgo.Context) (err error) {
			req := c.Request().(*Request)
			res := c.Response().(*Response)
			m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				err = next(c)
			})).ServeHTTP(res.ResponseWriter, req.Request)
			return
		}
	}
}
