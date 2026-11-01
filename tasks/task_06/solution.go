package main

import "container/list"

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}

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
	c := &LRUCache[K, V]{capacity: capacity}
	if capacity > 0 {
		c.items = make(map[K]*list.Element, capacity)
	}
	return c
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	elem, has := c.items[key]
	if !has {
		return
	}
	c.ll.MoveToFront(elem)
	return elem.Value.(*entry[K, V]).value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	if elem, has := c.items[key]; has {
		elem.Value.(*entry[K, V]).value = value
		return
	}
	if len(c.items) >= c.capacity {
		oldest := c.ll.Back()
		c.ll.Remove(oldest)
		delete(c.items, oldest.Value.(*entry[K, V]).key)
	}

	c.items[key] = c.ll.PushFront(&entry[K, V]{key: key, value: value})
}
