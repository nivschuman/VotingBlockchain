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

func (dialer *Dialer) Dial(ip net.IP, port uint16) error {
	address := net.JoinHostPort(ip.String(), fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}

	go dialer.ConnectionHandler(conn, true)

	return nil
}
