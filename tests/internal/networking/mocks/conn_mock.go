package networking_mocks

import (
	"bytes"
	"net"
	"time"
)

type ConnMock struct {
	*bytes.Buffer
}

func NewConnMock() *ConnMock {
	return &ConnMock{
		Buffer: new(bytes.Buffer),
	}
}

func (connMock *ConnMock) Read(p []byte) (n int, err error) {
	return connMock.Buffer.Read(p)
}

func (connMock *ConnMock) Write(p []byte) (n int, err error) {
	return connMock.Buffer.Write(p)
}

func (connMock *ConnMock) Close() error {
	// For a buffer, we don't really need to do anything for Close
	// But we need to satisfy the net.Conn interface
	return nil
}

func (c *ConnMock) LocalAddr() net.Addr {
	return nil
}

func (c *ConnMock) RemoteAddr() net.Addr {
	return nil
}

func (c *ConnMock) SetDeadline(t time.Time) error {
	return nil
}

func (c *ConnMock) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *ConnMock) SetWriteDeadline(t time.Time) error {
	return nil
}
