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
	if capacity < 0 {
		capacity = 0
	}

	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element, capacity),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	elem, ok := c.items[key]
	if !ok {
		return
	}

	c.ll.MoveToBack(elem)
	item := elem.Value.(entry[K, V])

	return item.value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}

	if e, ok := c.items[key]; ok {
		item := e.Value.(entry[K, V])
		item.value = value
		e.Value = item

		return
	}

	if len(c.items) >= c.capacity {
		old := c.ll.Front()
		item := old.Value.(entry[K, V])
		delete(c.items, item.key)
		c.ll.Remove(old)
	}

	elem := c.ll.PushBack(entry[K, V]{
		key:   key,
		value: value,
	})

	c.items[key] = elem
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
