package structures

import "bytes"

type BlockLocator struct {
	ids [][]byte
}

func NewBlockLocator() *BlockLocator {
	return &BlockLocator{
		ids: make([][]byte, 0),
	}
}

func (locator *BlockLocator) Get(idx int) []byte {
	return locator.ids[idx]
}

func (locator *BlockLocator) Add(blockId []byte) {
	locator.ids = append(locator.ids, blockId)
}

func (locator *BlockLocator) Length() int {
	return len(locator.ids)
}

func (locator *BlockLocator) Ids() [][]byte {
	copySlice := make([][]byte, len(locator.ids))
	for i, id := range locator.ids {
		idCopy := make([]byte, len(id))
		copy(idCopy, id)
		copySlice[i] = idCopy
	}
	return copySlice
}

func (locator *BlockLocator) AsBytes() ([]byte, error) {
	buf := new(bytes.Buffer)

	for _, id := range locator.ids {
		_, err := buf.Write(id)

		if err != nil {
			return nil, err
		}
	}

	return buf.Bytes(), nil
}
