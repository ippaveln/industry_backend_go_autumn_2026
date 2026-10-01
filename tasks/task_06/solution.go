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
	return &LRUCache[K, V]{capacity: capacity, items: make(map[K]*list.Element)}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	element, ok := c.items[key]
	if !ok {
		return
	}
	c.ll.MoveToFront(element)
	e := element.Value.(entry[K, V])
	return e.value, true
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	element, ok := c.items[key]
	if ok {
		e := element.Value.(entry[K, V])
		e.value = value
		element.Value = e
		return
	}
	if len(c.items) == c.capacity {
		element := c.ll.Back()
		e := element.Value.(entry[K, V])
		delete(c.items, e.key)
		c.ll.Remove(element)
	}
	e := entry[K, V]{key: key, value: value}
	element = c.ll.PushFront(e)
	c.items[key] = element
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
