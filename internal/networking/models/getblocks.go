package networking_models

import (
	"bytes"

	compact "github.com/nivschuman/VotingBlockchain/internal/networking/utils/compact"
	structures "github.com/nivschuman/VotingBlockchain/internal/structures"
)

type GetBlocks struct {
	BlockLocator *structures.BlockLocator
	StopHash     []byte
}

func NewGetBlocksMessage(getBlocks *GetBlocks) (*Message, error) {
	getBlocksBytes, err := getBlocks.AsBytes()

	if err != nil {
		return nil, err
	}

	return NewMessage(CommandGetBlocks, getBlocksBytes), nil
}

func NewGetBlocks(blockLocator *structures.BlockLocator, stopHash []byte) *GetBlocks {
	return &GetBlocks{
		BlockLocator: blockLocator,
		StopHash:     stopHash,
	}
}

func (getBlocks *GetBlocks) AsBytes() ([]byte, error) {
	var buf bytes.Buffer

	compactSize, err := compact.GetCompactSizeBytes(uint64(getBlocks.BlockLocator.Length()))
	if err != nil {
		return nil, err
	}

	buf.Write(compactSize)

	hashes, err := getBlocks.BlockLocator.AsBytes()
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(hashes)
	if err != nil {
		return nil, err
	}

	_, err = buf.Write(getBlocks.StopHash)
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func GetBlocksFromBytes(data []byte) (*GetBlocks, error) {
	buf := bytes.NewReader(data)
	compactSize, err := compact.ReadCompactSize(buf)
	if err != nil {
		return nil, err
	}

	blockLocator := structures.NewBlockLocator()
	for i := uint64(0); i < compactSize; i++ {
		hash := make([]byte, 32)
		if _, err := buf.Read(hash); err != nil {
			return nil, err
		}

		blockLocator.Add(hash)
	}

	stopHash := make([]byte, 32)
	if _, err := buf.Read(stopHash); err != nil {
		return nil, err
	}

	return NewGetBlocks(blockLocator, stopHash), nil
}
