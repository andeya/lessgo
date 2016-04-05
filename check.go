package lessgo

import (
	"net/http"
)

// 检查服务器是否启用
func checkServer() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			if !ServerEnable() {
				return c.NoContent(http.StatusServiceUnavailable)
			}
			return next(c)
		}
	}
}

// 检查是否为访问主页
func checkHome() MiddlewareFunc {
	return func(next HandlerFunc) HandlerFunc {
		return func(c Context) error {
			if c.Request().URL().Path() == "/" {
				c.Request().URL().SetPath(DefLessgo.home)
			}
			return next(c)
		}
	}
}
