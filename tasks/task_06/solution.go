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
	node := c.items[key]
	if node == nil {
		return value, false
	}
	c.ll.MoveToBack(node)
	return node.Value.(entry[K, V]).value, true
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	item := entry[K, V]{key: key, value: value}
	if node := c.items[key]; node != nil {
		node.Value = item
		return
	}
	if c.ll.Len() < c.capacity {
		c.items[key] = c.ll.PushBack(item)
		return
	}
	node := c.ll.Front()
	delete(c.items, node.Value.(entry[K, V]).key)
	node.Value = item
	c.ll.MoveToBack(node)
	c.items[key] = node
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
