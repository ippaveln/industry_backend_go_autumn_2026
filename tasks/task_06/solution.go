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
		ll:       list.List{},
		items:    make(map[K]*list.Element, capacity),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	elem, ok := c.items[key]
	if ok {
		value = elem.Value.(entry[K, V]).value
		c.ll.MoveToFront(elem)
	}
	return value, ok
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	if elem, ok := c.items[key]; ok {
		elem.Value = entry[K, V]{key, value}
		return
	}
	if len(c.items) == c.capacity {
		last := c.ll.Back()
		c.ll.Remove(last)
		delete(c.items, last.Value.(entry[K, V]).key)
	}
	elem := c.ll.PushFront(entry[K, V]{key, value})
	c.items[key] = elem
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
