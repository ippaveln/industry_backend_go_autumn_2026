package main

import "container/list"

type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items: make(map[K]*list.Element),
	}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.capacity <= 0 {
		var zero V
		return zero, false
	}

	if item, exists := c.items[key]; exists {
		c.ll.MoveToFront(item)
		return item.Value.(*entry[K, V]).value, true
	}

	var zero V
	return zero, false
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}

	if item, exists := c.items[key]; exists {
		item.Value.(*entry[K, V]).value = value
		return
	}

	if len(c.items) >= c.capacity {
		item := c.ll.Back()
		if item != nil {
			c.ll.Remove(item)
			e := item.Value.(*entry[K, V])
			delete(c.items, e.key)
		}
	}

	e := &entry[K, V]{key: key, value: value}
	item := c.ll.PushFront(e)
	c.items[key] = item
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
