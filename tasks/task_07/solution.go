package main

import (
	"container/list"
	"sync"
)

type LRU[K comparable, V any] interface {
	Get(key K) (value V, ok bool)
	Set(key K, value V)
}
type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	m := make(map[K]*list.Element)
	cache := LRUCache[K, V]{
		capacity: capacity,
		items:    m,
	}
	return &cache
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, found := c.items[key]
	if !found {
		return
	}
	c.ll.MoveToFront(e)
	value = e.Value.(entry[K, V]).value
	return value, true
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.capacity <= 0 {
		return
	}
	e, ok := c.items[key]
	if ok {
		e.Value = entry[K, V]{key: key, value: value}
	} else {
		if len(c.items) >= c.capacity {
			last_node := c.ll.Back()
			delete(c.items, last_node.Value.(entry[K, V]).key)
			c.ll.Remove(last_node)
		}
		key_new := c.ll.PushFront(entry[K, V]{key: key, value: value})
		c.items[key] = key_new
	}
}
