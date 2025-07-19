package ip

import (
	"fmt"
	"net"
	"strconv"
)

func ConnToIpAndPort(conn net.Conn) (net.IP, uint16, error) {
	host, portStr, err := net.SplitHostPort(conn.RemoteAddr().String())
	if err != nil {
		return nil, 0, err
	}

	ip := net.ParseIP(host)
	if ip == nil {
		return nil, 0, fmt.Errorf("invalid IP: %s", host)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil || port < 0 || port > 65535 {
		return nil, 0, fmt.Errorf("invalid port: %s", portStr)
	}

	return ip, uint16(port), nil
}
