package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	items := make(map[K]V, capacity)
	return &Cache[K, V]{
		capacity: capacity,
		items:    items,
	}
}

func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	v, ok = c.items[k]
	return v, ok
}

func (c *Cache[K, V]) Set(k K, v V) bool {
	if _, ok := c.items[k]; ok {
		c.items[k] = v
		return true
	}
	if len(c.items) >= c.capacity {
		return false
	}
	c.items[k] = v
	return true
}
