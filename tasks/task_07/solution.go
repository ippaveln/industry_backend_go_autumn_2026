package main

import (
	"container/list"
	"sync"
)

type LRU[K comparable, V any] interface {
	Get(key K) (value V, ok bool)
	Set(key K, value V)
}
type entry[V any] struct {
	value   V
	element *list.Element
}

type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	ll       list.List
	items    map[K]entry[V]
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]entry[V]),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if !ok {
		return value, false
	}
	c.ll.MoveToFront(item.element)
	return item.value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}
	if item, ok := c.items[key]; ok {
		item.value = value
		c.items[key] = item
		return
	}
	if len(c.items) == c.capacity {
		oldest := c.ll.Back()
		// A nil interface list value yields zero K when the assertion fails.
		oldestKey, _ := oldest.Value.(K)
		delete(c.items, oldestKey)
		c.ll.Remove(oldest)
	}
	element := c.ll.PushFront(key)
	c.items[key] = entry[V]{value: value, element: element}
}
