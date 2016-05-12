package lessgo

type (
	// Group is a set of sub-routes for a specified route. It can be used for inner
	// routes that share a common middlware or functionality that should be separate
	// from the parent echo instance while still inheriting from it.
	Group struct {
		prefix     string
		middleware []MiddlewareFunc
		echo       *Echo
	}
)

// Use implements `Echo#Use()` for sub-routes within the Group.
func (g *Group) Use(m ...MiddlewareFunc) {
	g.middleware = append(g.middleware, m...)
}

// Connect implements `Echo#Connect()` for sub-routes within the Group.
func (g *Group) Connect(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(CONNECT, path, h, m...)
}

// Delete implements `Echo#Delete()` for sub-routes within the Group.
func (g *Group) Delete(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(DELETE, path, h, m...)
}

// Get implements `Echo#Get()` for sub-routes within the Group.
func (g *Group) Get(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(GET, path, h, m...)
}

// Head implements `Echo#Head()` for sub-routes within the Group.
func (g *Group) Head(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(HEAD, path, h, m...)
}

// Options implements `Echo#Options()` for sub-routes within the Group.
func (g *Group) Options(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(OPTIONS, path, h, m...)
}

// Patch implements `Echo#Patch()` for sub-routes within the Group.
func (g *Group) Patch(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(PATCH, path, h, m...)
}

// Post implements `Echo#Post()` for sub-routes within the Group.
func (g *Group) Post(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(POST, path, h, m...)
}

// Put implements `Echo#Put()` for sub-routes within the Group.
func (g *Group) Put(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(PUT, path, h, m...)
}

// Trace implements `Echo#Trace()` for sub-routes within the Group.
func (g *Group) Trace(path string, h HandlerFunc, m ...MiddlewareFunc) {
	g.add(TRACE, path, h, m...)
}

// Any implements `Echo#Any()` for sub-routes within the Group.
func (g *Group) Any(path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	for _, m := range methods {
		g.add(m, path, handler, middleware...)
	}
}

// Match implements `Echo#Match()` for sub-routes within the Group.
func (g *Group) Match(methods []string, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	for _, method := range methods {
		g.add(method, path, handler, middleware...)
	}
}

// Group creates a new sub-group with prefix and optional sub-group-level middleware.
func (g *Group) Group(prefix string, m ...MiddlewareFunc) *Group {
	m = append(g.middleware, m...)
	return g.echo.Group(joinpath(g.prefix, prefix), m...)
}

func (g *Group) add(methods, path string, handler HandlerFunc, middleware ...MiddlewareFunc) {
	path = joinpath(g.prefix, path)
	middleware = append(g.middleware, middleware...)
	switch methods {
	case WS:
		g.echo.WebSocket(path, handler, middleware...)
	default:
		g.echo.add(methods, path, handler, middleware...)
	}
}
