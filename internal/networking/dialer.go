package networking

import (
	"fmt"
	"net"
)

type Dialer struct {
	IP                string
	Port              int
	ConnectionHandler ConnectionHandler
}

func (dialer *Dialer) Dial() error {
	address := fmt.Sprintf("%s:%d", dialer.IP, dialer.Port)

	conn, err := net.Dial("tcp", address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %v", address, err)
	}

	go dialer.ConnectionHandler(conn)

	return nil
}
