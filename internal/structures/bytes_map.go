package structures

type BytesMap[V any] struct {
	data map[string]V
}

func NewBytesMap[V any]() *BytesMap[V] {
	return &BytesMap[V]{
		data: make(map[string]V),
	}
}

func (m *BytesMap[V]) Put(key []byte, value V) {
	m.data[string(key)] = value
}

func (m *BytesMap[V]) Get(key []byte) (V, bool) {
	val, exists := m.data[string(key)]
	return val, exists
}

func (m *BytesMap[V]) ContainsKey(key []byte) bool {
	_, exists := m.data[string(key)]
	return exists
}

func (m *BytesMap[V]) Remove(key []byte) {
	delete(m.data, string(key))
}

func (m *BytesMap[V]) Keys() [][]byte {
	keys := make([][]byte, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, []byte(k))
	}
	return keys
}

func (m *BytesMap[V]) Values() []V {
	values := make([]V, 0, len(m.data))
	for _, v := range m.data {
		values = append(values, v)
	}
	return values
}

func (m *BytesMap[V]) Length() int {
	return len(m.data)
}
