package networking_models

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"

	compact "github.com/nivschuman/VotingBlockchain/internal/networking/utils/compact"
)

const MAX_ADDR_SIZE = 1000

type Address struct {
	Ip       net.IP //Ip of address, 16 byte ipv6 format
	Port     uint16 //Port of address
	NodeType uint32 //Type of node related to address
}

type Addr struct {
	Count     uint64
	Addresses []*Address
}

func NewAddr() *Addr {
	return &Addr{
		Count:     0,
		Addresses: make([]*Address, 0),
	}
}

func NewAddrMessage(addr *Addr) (*Message, error) {
	addrBytes, err := addr.AsBytes()
	if err != nil {
		return nil, err
	}

	return NewMessage(CommandAddr, addrBytes), nil
}

func AddrFromBytes(b []byte) (*Addr, error) {
	buf := bytes.NewReader(b)

	compactSize, err := compact.ReadCompactSize(buf)
	if err != nil {
		return nil, err
	}

	addr := NewAddr()
	for range compactSize {
		address := &Address{}

		ipBytes := make([]byte, 16)
		_, err = buf.Read(ipBytes)
		if err != nil {
			return nil, err
		}
		address.Ip = net.IP(ipBytes)

		err = binary.Read(buf, binary.BigEndian, address.Port)
		if err != nil {
			return nil, err
		}

		err = binary.Read(buf, binary.BigEndian, address.NodeType)
		if err != nil {
			return nil, err
		}

		addr.AddAddress(address)
	}

	return addr, nil
}

func (addr *Addr) AddAddress(address *Address) {
	addr.Addresses = append(addr.Addresses, address)
}

func (addr *Addr) AsBytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	compactSize, err := compact.GetCompactSizeBytes(addr.Count)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(compactSize)
	if err != nil {
		return nil, err
	}

	for _, address := range addr.Addresses {
		err = binary.Write(buf, binary.BigEndian, address.Ip.To16())
		if err != nil {
			return nil, err
		}

		err = binary.Write(buf, binary.BigEndian, address.Port)
		if err != nil {
			return nil, err
		}

		err = binary.Write(buf, binary.BigEndian, address.NodeType)
		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (address *Address) IsValid() bool {
	// Check port is in valid range (1–65535)
	if address.Port == 0 {
		return false
	}

	// Check IP is not nil and is a valid IPv4 or IPv6
	if address.Ip == nil || (address.Ip.To4() == nil && address.Ip.To16() == nil) {
		return false
	}

	return true
}

func (address *Address) IsRoutable() bool {
	if address.Ip == nil || address.Ip.IsUnspecified() || address.Ip.IsLoopback() || address.Ip.IsMulticast() || address.Ip.IsLinkLocalUnicast() || address.Ip.IsLinkLocalMulticast() || address.Ip.IsPrivate() {
		return false
	}

	ip4 := address.Ip.To4()
	if ip4 != nil {
		// 100.64.0.0/10 — Carrier-grade NAT (RFC 6598)
		if ip4[0] == 100 && (ip4[1]&0b11000000) == 64 {
			return false
		}

		// 192.0.2.0/24 (TEST-NET-1)
		if ip4[0] == 192 && ip4[1] == 0 && ip4[2] == 2 {
			return false
		}

		// 198.51.100.0/24 (TEST-NET-2)
		if ip4[0] == 198 && ip4[1] == 51 && ip4[2] == 100 {
			return false
		}

		// 203.0.113.0/24 (TEST-NET-3)
		if ip4[0] == 203 && ip4[1] == 0 && ip4[2] == 113 {
			return false
		}

		// 198.18.0.0/15 — benchmarking (RFC 2544)
		if ip4[0] == 198 && (ip4[1] == 18 || ip4[1] == 19) {
			return false
		}

		return true
	}

	// IPv6 checks
	ip16 := address.Ip.To16()
	if ip16 == nil {
		return false
	}

	// RFC 4862 — autoconfiguration
	if ip16[0] == 0xfe && (ip16[1]&0xc0) == 0x80 {
		return false
	}

	// RFC 4193 — unique local addresses (fc00::/7)
	if (ip16[0] & 0xfe) == 0xfc {
		return false
	}

	// RFC 4843 — ORCHID (Overlay Routable Cryptographic Hash Identifiers) (2001:10::/28)
	if ip16[0] == 0x20 && ip16[1] == 0x01 && ip16[2] == 0x00 && (ip16[3]&0xf0) == 0x10 {
		return false
	}

	// RFC 7343 — ORCHIDv2 (2001:20::/28)
	if ip16[0] == 0x20 && ip16[1] == 0x01 && ip16[2] == 0x00 && (ip16[3]&0xf0) == 0x20 {
		return false
	}

	return true
}

func (address *Address) String() string {
	return fmt.Sprintf("Address(IP=%s, Port=%d, NodeType=%d)", address.Ip.String(), address.Port, address.NodeType)
}
