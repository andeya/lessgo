package middleware

import (
	"encoding/base64"

	"github.com/lessgo/lessgo"
)

type (
	// BasicAuthConfig defines the config for HTTP basic auth middleware.
	BasicAuthConfig struct {
		// AuthFunc is the function to validate basic auth credentials.
		AuthFunc BasicAuthFunc
	}

	// BasicAuthFunc defines a function to validate basic auth credentials.
	BasicAuthFunc func(string, string) bool
)

const (
	basic = "Basic"
)

var (
	// DefaultBasicAuthConfig is the default basic auth middleware config.
	DefaultBasicAuthConfig = BasicAuthConfig{}
)

// BasicAuth returns an HTTP basic auth middleware.
//
// For valid credentials it calls the next handler.
// For invalid credentials, it sends "401 - Unauthorized" response.
func BasicAuth(f BasicAuthFunc) lessgo.MiddlewareFunc {
	c := DefaultBasicAuthConfig
	c.AuthFunc = f
	return BasicAuthFromConfig(c)
}

// BasicAuthFromConfig returns an HTTP basic auth middleware from config.
// See `BasicAuth()`.
func BasicAuthFromConfig(config BasicAuthConfig) lessgo.MiddlewareFunc {
	return func(next lessgo.HandlerFunc) lessgo.HandlerFunc {
		return func(c lessgo.Context) error {
			auth := c.Request().Header().Get(lessgo.Authorization)
			l := len(basic)

			if len(auth) > l+1 && auth[:l] == basic {
				b, err := base64.StdEncoding.DecodeString(auth[l+1:])
				if err == nil {
					cred := string(b)
					for i := 0; i < len(cred); i++ {
						if cred[i] == ':' {
							// Verify credentials
							if config.AuthFunc(cred[:i], cred[i+1:]) {
								return next(c)
							}
						}
					}
				}
			}
			c.Response().Header().Set(lessgo.WWWAuthenticate, basic+" realm=Restricted")
			return lessgo.ErrUnauthorized
		}
	}
}
