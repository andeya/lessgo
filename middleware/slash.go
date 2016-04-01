package middleware

import (
	"github.com/lessgo/lessgo"
)

// AddTrailingSlash returns a root level (before router) middleware which adds a
// trailing slash to the request `URL#Path`.
//
// Usage `Echo#Pre(AddTrailingSlash())`
func AddTrailingSlash() lessgo.MiddlewareFunc {
	return func(next lessgo.Handler) lessgo.Handler {
		return lessgo.HandlerFunc(func(c lessgo.Context) error {
			url := c.Request().URL()
			path := url.Path()
			if path != "/" && path[len(path)-1] != '/' {
				url.SetPath(path + "/")
			}
			return next.Handle(c)
		})
	}
}

// RemoveTrailingSlash returns a root level (before router) middleware which removes
// a trailing slash from the request URI.
//
// Usage `Echo#Pre(RemoveTrailingSlash())`
func RemoveTrailingSlash() lessgo.MiddlewareFunc {
	return func(next lessgo.Handler) lessgo.Handler {
		return lessgo.HandlerFunc(func(c lessgo.Context) error {
			url := c.Request().URL()
			path := url.Path()
			l := len(path) - 1
			if path != "/" && path[l] == '/' {
				url.SetPath(path[:l])
			}
			return next.Handle(c)
		})
	}
}
