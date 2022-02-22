package ratelimiter

import (
	"net"
)

type stubConn struct {
	net.Conn
	remoteAddr net.Addr
}

func (s stubConn) RemoteAddr() net.Addr {
	return s.remoteAddr
}
