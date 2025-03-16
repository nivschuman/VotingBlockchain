package networking_connector

import (
	"fmt"
	"net"
)

type Dialer struct {
	ConnectionHandler ConnectionHandler
}

func NewDialer() *Dialer {
	return &Dialer{}
}

func (dialer *Dialer) Dial(ip string, port uint16) error {
	address := net.JoinHostPort(ip, fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", address, err)
	}

	go dialer.ConnectionHandler(conn, true)

	return nil
}
