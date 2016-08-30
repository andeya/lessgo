package lessgo

import (
	"sync"

	"github.com/lessgo/lessgo/utils"
)

// Router is a http.Handler which can be used to dispatch requests to different
// handler functions via configurable routes
type Router struct {
	trees map[string]*node

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 307 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 307 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	// If enabled, the router automatically replies to OPTIONS requests.
	// Custom OPTIONS handlers take priority over automatic replies.
	HandleOPTIONS bool

	sync.RWMutex
}

// NewRouter returns a new initialized Router.
// Path auto-correction, including trailing slashes, is enabled by default.
func newRouter() *Router {
	return &Router{
		RedirectTrailingSlash:  true,
		RedirectFixedPath:      true,
		HandleMethodNotAllowed: true,
		HandleOPTIONS:          true,
	}
}

// Handle registers a new request handle with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) Handle(method, path string, handle HandlerFunc) {
	if path[0] != '/' {
		panic("path must begin with '/' in path '" + path + "'")
	}

	if r.trees == nil {
		r.trees = make(map[string]*node)
	}

	root := r.trees[method]
	if root == nil {
		root = new(node)
		r.trees[method] = root
	}

	root.addRoute(path, handle)
}

func (r *Router) allowed(path, reqMethod string, pkeys, pvalues []string) string {
	var allow string
	if path == "*" { // server-wide
		for method := range r.trees {
			if method == OPTIONS {
				continue
			}

			// add request method to list of allowed methods
			if len(allow) == 0 {
				allow = method
			} else {
				allow += ", " + method
			}
		}
	} else { // specific path
		for method := range r.trees {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == OPTIONS {
				continue
			}

			handle, _, _, _ := r.trees[method].getValue(path, pkeys, pvalues)
			if handle != nil {
				// add request method to list of allowed methods
				if len(allow) == 0 {
					allow = method
				} else {
					allow += ", " + method
				}
			}
		}
	}
	if len(allow) > 0 {
		allow += ", OPTIONS"
	}
	return allow
}

// ServeHTTP makes the router implement the MiddlewareFunc.
func (r *Router) process(next HandlerFunc) HandlerFunc {
	return func(c *Context) error {
		var req = c.request

		r.RLock()
		var root = r.trees[req.Method]
		r.RUnlock()

		var path = req.URL.Path

		if root != nil {
			var handle HandlerFunc
			var tsr bool
			handle, c.pkeys, c.pvalues, tsr = root.getValue(path, c.pkeys, c.pvalues)
			if handle != nil {
				if err := handle(c); err != nil {
					return err
				}
				return next(c)
			} else if req.Method != CONNECT && path != "/" {
				code := 301 // Permanent redirect, request with GET method
				if req.Method != GET {
					// Temporary redirect, request with same method
					// As of Go 1.3, Go does not support status code 308.
					code = 307
				}

				if tsr && r.RedirectTrailingSlash {
					if len(path) > 1 && path[len(path)-1] == '/' {
						req.URL.Path = path[:len(path)-1]
					} else {
						req.URL.Path = path + "/"
					}
					return c.Redirect(code, req.URL.String())
				}

				// Try to fix the request path
				if r.RedirectFixedPath {
					fixedPath, found := root.findCaseInsensitivePath(
						CleanPath(path),
						r.RedirectTrailingSlash,
					)
					if found {
						req.URL.Path = utils.Bytes2String(fixedPath)
						return c.Redirect(code, req.URL.String())
					}
				}
			}
		}

		if req.Method == OPTIONS {
			// Handle OPTIONS requests
			if r.HandleOPTIONS {
				if allow := r.allowed(path, req.Method, c.pkeys, c.pvalues); len(allow) > 0 {
					c.response.Header().Set("Allow", allow)
					return c.NoContent(200)
				}
			}
		} else {
			// Handle 405
			if r.HandleMethodNotAllowed {
				if allow := r.allowed(path, req.Method, c.pkeys, c.pvalues); len(allow) > 0 {
					c.response.Header().Set("Allow", allow)
					return c.Failure(405, nil)
				}
			}
		}

		// Handle 404
		return c.Failure(404, nil)
	}
}
