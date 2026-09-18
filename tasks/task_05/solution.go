package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	panic("TODO: implement")
}
func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	panic("TODO: implement")
}
func (c *Cache[K, V]) Set(k K, v V) bool {
	panic("TODO: implement")
}
