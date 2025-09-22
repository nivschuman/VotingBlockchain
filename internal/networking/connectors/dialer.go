package networking_connector

import (
	"context"
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

func (dialer *Dialer) DialContext(ip net.IP, port uint16, ctx context.Context) error {
	address := net.JoinHostPort(ip.String(), fmt.Sprint(port))
	netDialer := &net.Dialer{}

	conn, err := netDialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return err
	}

	go dialer.connectionHandler(conn, true)
	return nil
}
