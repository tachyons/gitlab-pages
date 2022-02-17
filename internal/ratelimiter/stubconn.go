package ratelimiter

import (
	"net"
	"time"
)

type stubConn struct {
	remoteAddr net.Addr
}

func (s stubConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (s stubConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (s stubConn) Close() error {
	return nil
}

func (s stubConn) LocalAddr() net.Addr {
	return &net.IPAddr{
		IP:   net.IPv4(10, 10, 10, 10),
		Zone: "",
	}
}

func (s stubConn) RemoteAddr() net.Addr {
	return s.remoteAddr
}

func (s stubConn) SetDeadline(t time.Time) error {
	return nil
}

func (s stubConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (s stubConn) SetWriteDeadline(t time.Time) error {
	return nil
}
