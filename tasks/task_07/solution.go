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
		items:    map[K]*list.Element{},
	}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {

	c.mu.Lock()
	defer c.mu.Unlock()

	e, found := c.items[key]

	if !found {
		return // default type value / false
	}

	c.ll.MoveToFront(e)

	ent := e.Value.(entry[K, V])
	return ent.value, true

}
func (c *LRUCache[K, V]) Set(key K, value V) {

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.capacity <= 0 {
		return
	}

	if e, found := c.items[key]; found {
		e.Value = entry[K, V]{key: key, value: value}
		return
	}

	if c.ll.Len() >= c.capacity {

		obj := c.ll.Back()
		objKey := obj.Value.(entry[K, V]).key
		c.ll.Remove(obj)
		delete(c.items, objKey)
	}

	ent := entry[K, V]{
		key:   key,
		value: value,
	}

	e := c.ll.PushFront(ent)
	c.items[key] = e

}
