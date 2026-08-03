package xmlmapper

// strictMap only admits a key once. Keeping the backing map private prevents
// callers from accidentally replacing an earlier definition.
type strictMap[V any] struct {
	entries map[string]V
}

func newStrictMap[V any]() strictMap[V] {
	return strictMap[V]{entries: make(map[string]V)}
}

func (m strictMap[V]) add(key string, value V) bool {
	if _, exists := m.entries[key]; exists {
		return false
	}
	m.entries[key] = value
	return true
}

func (m strictMap[V]) get(key string) (V, bool) {
	value, exists := m.entries[key]
	return value, exists
}
