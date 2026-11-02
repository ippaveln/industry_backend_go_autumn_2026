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
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
	}
}
func (c *LRUCache[K, V]) Get(key K) (value V, exists bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, exists := c.items[key]
	if !exists {
		return
	}
	c.ll.MoveToBack(e)
	return e.Value.(*entry[K, V]).value, exists
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	e, exists := c.items[key]
	if exists {
		e.Value.(*entry[K, V]).value = value
		return
	}
	if c.capacity <= len(c.items) {
		oldestNode := c.ll.Front()
		c.ll.Remove(oldestNode)
		delete(c.items, oldestNode.Value.(*entry[K, V]).key)
	}

	newEntry := &entry[K, V]{key: key, value: value}
	newElement := c.ll.PushBack(newEntry)
	c.items[key] = newElement
	return
}
