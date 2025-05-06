package compact

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

func GetCompactSizeBytes(size uint64) ([]byte, error) {
	var buf bytes.Buffer

	if size < 0xFD {
		if err := binary.Write(&buf, binary.BigEndian, uint8(size)); err != nil {
			return nil, err
		}
	} else if size <= 0xFFFF {
		if err := binary.Write(&buf, binary.BigEndian, uint8(0xFD)); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, uint16(size)); err != nil {
			return nil, err
		}
	} else if size <= 0xFFFFFFFF {
		if err := binary.Write(&buf, binary.BigEndian, uint8(0xFE)); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, uint32(size)); err != nil {
			return nil, err
		}
	} else {
		if err := binary.Write(&buf, binary.BigEndian, uint8(0xFF)); err != nil {
			return nil, err
		}
		if err := binary.Write(&buf, binary.BigEndian, uint64(size)); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func ReadCompactSize(buf *bytes.Reader) (uint64, error) {
	var size uint64
	var byte1 byte
	if err := binary.Read(buf, binary.BigEndian, &byte1); err != nil {
		return 0, err
	}

	switch {
	case byte1 < 0xFD:
		size = uint64(byte1)
	case byte1 == 0xFD:
		var byte2 uint16
		if err := binary.Read(buf, binary.BigEndian, &byte2); err != nil {
			return 0, err
		}
		size = uint64(byte2)
	case byte1 == 0xFE:
		var byte4 uint32
		if err := binary.Read(buf, binary.BigEndian, &byte4); err != nil {
			return 0, err
		}
		size = uint64(byte4)
	case byte1 == 0xFF:
		var byte8 uint64
		if err := binary.Read(buf, binary.BigEndian, &byte8); err != nil {
			return 0, err
		}
		size = byte8
	default:
		return 0, fmt.Errorf("invalid byte for compact size encoding: %x", byte1)
	}

	return size, nil
}
