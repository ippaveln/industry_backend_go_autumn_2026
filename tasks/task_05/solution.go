package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	return &Cache[K, V]{
		items: make(map[K]V),
		capacity: capacity,
	}
}
func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	if c.capacity <= 0 {
		var zero V
		return zero, false
	}

	val, exists := c.items[k]
	return val, exists
}
func (c *Cache[K, V]) Set(k K, v V) bool {
	if c.capacity <= 0 {
		return false
	}

	if _, exists := c.items[k]; exists {
		c.items[k] = v
		return true
	}

	if len(c.items) >= c.capacity {
		return false
	}

	c.items[k] = v
	return true
}
