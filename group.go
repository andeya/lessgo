package lessgo

type (
	// Group is a set of sub-routes for a specified route. It can be used for inner
	// routes that share a common middlware or functionality that should be separate
	// from the parent app instance while still inheriting from it.
	Group struct {
		prefix     string
		chainNodes []MiddlewareFunc
		app        *App
	}
)

// Group creates a new sub-group with prefix and optional sub-group-level middleware.
func (g *Group) group(prefix string, m ...MiddlewareFunc) *Group {
	m = append(g.chainNodes, m...)
	return g.app.group(joinpath(g.prefix, prefix), m...)
}

// Use implements `App#Use()` for sub-routes within the Group.
func (g *Group) use(m ...MiddlewareFunc) {
	g.chainNodes = append(g.chainNodes, m...)
}

// match implements `App#match()` for sub-routes within the Group.
func (g *Group) match(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	for _, method := range methods {
		g.add(method, path, handler, middleware...)
	}
}

func (g *Group) add(methods, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	path = joinpath(g.prefix, path)
	middleware = append(g.chainNodes, middleware...)
	switch methods {
	case WS:
		g.app.webSocket(path, handler, middleware...)
	default:
		g.app.add(methods, path, handler, middleware...)
	}
}
