package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	m := make(map[K]V)
	cache := Cache[K, V]{
		capacity: capacity,
		items:    m,
	}
	return &cache
}

func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	v, ok = c.items[k]
	return v, ok
}
func (c *Cache[K, V]) Set(k K, v V) bool {
	if c.capacity <= 0 {
		return false
	}
	_, ok := c.Get(k)
	if ok {
		c.items[k] = v
		return true
	} else {
		if len(c.items) == c.capacity {
			return false
		} else {
			c.items[k] = v
		}
	}
	return true
}
