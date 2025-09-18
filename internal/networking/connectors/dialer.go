package networking_connector

import (
	"fmt"
	"net"
)

type Dialer struct {
	connectionHandler ConnectionHandler
}

func NewDialer(connectionHandler ConnectionHandler) *Dialer {
	return &Dialer{connectionHandler: connectionHandler}
}

func (dialer *Dialer) Dial(ip net.IP, port uint16) error {
	address := net.JoinHostPort(ip.String(), fmt.Sprint(port))

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}

	go dialer.connectionHandler(conn, true)

	return nil
}
