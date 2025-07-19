package networking_models

import (
	"bytes"
	"encoding/binary"

	compact "github.com/nivschuman/VotingBlockchain/internal/networking/utils/compact"
)

const MSG_TX = uint32(1)
const MSG_BLOCK = uint32(2)

type InvItem struct {
	Type uint32
	Hash []byte
}

type Inv struct {
	Count uint64
	Items []InvItem
}

func NewInv() *Inv {
	return &Inv{
		Count: 0,
		Items: make([]InvItem, 0),
	}
}

func NewInvMessage(inv *Inv) (*Message, error) {
	invBytes, err := inv.AsBytes()

	if err != nil {
		return nil, err
	}

	return NewMessage(CommandInv, invBytes), nil
}

func (inv *Inv) AddItem(itemType uint32, itemHash []byte) {
	invItem := InvItem{
		Type: itemType,
		Hash: itemHash,
	}

	inv.Items = append(inv.Items, invItem)
	inv.Count++
}

func (inv *Inv) AsBytes() ([]byte, error) {
	var buf bytes.Buffer

	compactSize, err := compact.GetCompactSizeBytes(inv.Count)

	if err != nil {
		return nil, err
	}

	if _, err := buf.Write(compactSize); err != nil {
		return nil, err
	}

	for _, item := range inv.Items {
		if err := binary.Write(&buf, binary.BigEndian, item.Type); err != nil {
			return nil, err
		}

		if _, err := buf.Write(item.Hash); err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}

func (inv *Inv) Clear() {
	inv.Items = make([]InvItem, 0)
	inv.Count = 0
}

func (inv *Inv) Contains(itemType uint32, itemHash []byte) bool {
	for _, item := range inv.Items {
		if item.Type == itemType && bytes.Equal(item.Hash, itemHash) {
			return true
		}
	}
	return false
}

func InvFromBytes(data []byte) (*Inv, error) {
	buf := bytes.NewReader(data)

	compactSize, err := compact.ReadCompactSize(buf)
	if err != nil {
		return nil, err
	}

	inv := NewInv()

	for i := uint64(0); i < compactSize; i++ {
		var itemType uint32

		if err := binary.Read(buf, binary.BigEndian, &itemType); err != nil {
			return nil, err
		}

		hash := make([]byte, 32)
		if _, err := buf.Read(hash); err != nil {
			return nil, err
		}

		inv.AddItem(itemType, hash)
	}

	return inv, nil
}
