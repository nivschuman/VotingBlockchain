package structures

type BytesSet struct {
	set map[string]struct{}
}

func NewBytesSet() *BytesSet {
	return &BytesSet{
		set: make(map[string]struct{}),
	}
}

func (bytesSet *BytesSet) Add(bytes []byte) {
	bytesSet.set[string(bytes)] = struct{}{}
}

func (bytesSet *BytesSet) Contains(bytes []byte) bool {
	_, exists := bytesSet.set[string(bytes)]
	return exists
}

func (bytesSet *BytesSet) Remove(bytes []byte) {
	delete(bytesSet.set, string(bytes))
}

func (bytesSet *BytesSet) ToBytesSlice() [][]byte {
	var result [][]byte
	for key := range bytesSet.set {
		result = append(result, []byte(key))
	}
	return result
}

func (bytesSet *BytesSet) Length() int {
	return len(bytesSet.set)
}
