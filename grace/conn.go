package grace

import (
	"net"
)

type graceConn struct {
	net.Conn
	server *Server
}

func (c graceConn) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = c.server.fixPanic(r)
		}
	}()
	c.server.wg.Done()
	return c.Conn.Close()
}
