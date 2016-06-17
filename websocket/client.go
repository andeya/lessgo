// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package websocket

import (
	"bufio"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
)

// DialError is an error that occurs while dialling a websocket server.
type DialError struct {
	*Config
	Err error
}

func (e *DialError) Error() string {
	return "websocket.Dial " + e.Config.Location.String() + ": " + e.Err.Error()
}

// NewConfig creates a new WebSocket config for client connection.
func NewConfig(server, origin string) (*Config, error) {
	var err error
	config := new(Config)
	config.Version = ProtocolVersionHybi13
	config.Location, err = url.ParseRequestURI(server)
	if err != nil {
		return nil, err
	}
	config.Origin, err = url.ParseRequestURI(origin)
	if err != nil {
		return nil, err
	}
	config.Header = http.Header(make(map[string][]string))
	return config, nil
}

// NewClient creates a new WebSocket client connection over rwc.
func NewClient(config *Config, rwc io.ReadWriteCloser) (*Conn, error) {
	br := bufio.NewReader(rwc)
	bw := bufio.NewWriter(rwc)
	err := hybiClientHandshake(config, br, bw)
	if err != nil {
		return nil, err
	}
	buf := bufio.NewReadWriter(br, bw)
	return newHybiClientConn(config, buf, rwc), nil
}

// Dial opens a new client connection to a WebSocket.
func Dial(url_, protocol, origin string) (*Conn, error) {
	config, err := NewConfig(url_, origin)
	if err != nil {
		return nil, err
	}
	if protocol != "" {
		config.Protocol = []string{protocol}
	}
	return DialConfig(config)
}

// DialConfig opens a new client connection to a WebSocket with a config.
func DialConfig(config *Config) (*Conn, error) {
	var client net.Conn
	if config.Location == nil {
		return nil, &DialError{config, ErrBadWebSocketLocation}
	}
	if config.Origin == nil {
		return nil, &DialError{config, ErrBadWebSocketOrigin}
	}
	var (
		ws  *Conn
		err error
	)

	switch config.Location.Scheme {
	case "ws":
		client, err = net.Dial("tcp", config.Location.Host)

	case "wss":
		client, err = tls.Dial("tcp", config.Location.Host, config.TlsConfig)

	default:
		err = ErrBadScheme
	}
	if err != nil {
		goto Error
	}

	ws, err = NewClient(config, client)
	if err != nil {
		goto Error
	}
	return ws, nil

Error:
	return nil, &DialError{config, err}
}
