package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	c := &Cache[K, V]{capacity: capacity}
	// При capacity <= 0 карта остаётся nil: читать из неё можно,
	// а запись невозможна по построению — Set до неё не доходит.
	if capacity > 0 {
		c.items = make(map[K]V, capacity)
	}

	return c
}

func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	v, ok = c.items[k]

	return v, ok
}

func (c *Cache[K, V]) Set(k K, v V) bool {
	if _, exists := c.items[k]; !exists && len(c.items) >= c.capacity {
		return false
	}
	c.items[k] = v

	return true
}
