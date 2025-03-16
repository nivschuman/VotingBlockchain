package ip

import (
	"encoding/binary"
	"fmt"
	"net"
)

func Uint32ToIPv4(ipUint32 uint32) string {
	ipBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(ipBytes, ipUint32)
	return net.IP(ipBytes).String()
}

func Ipv4ToUint32(ipStr string) (uint32, error) {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return 0, fmt.Errorf("invalid IP address: %s", ipStr)
	}

	ip = ip.To4()
	if ip == nil {
		return 0, fmt.Errorf("not a valid IPv4 address: %s", ipStr)
	}

	return binary.BigEndian.Uint32(ip), nil
}
