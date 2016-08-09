package grace

import (
	"errors"
	"net"
)

type graceConn struct {
	net.Conn
	server *Server
}

func (c graceConn) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			switch x := r.(type) {
			case string:
				if x == "sync: negative WaitGroup counter" {
					c.server.wg.Add(1)
				} else {
					err = errors.New(x)
				}
			case error:
				err = x
			default:
				err = errors.New("Unknown panic")
			}
		}
	}()
	c.server.wg.Done()
	return c.Conn.Close()
}
