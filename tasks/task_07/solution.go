package main

import (
	"container/list"
	"sync"
)

type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	ll       list.List
	items    map[K]*list.Element
	mutex    sync.Mutex
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	return &LRUCache[K, V]{capacity: capacity, ll: list.List{}, items: make(map[K]*list.Element, capacity)}
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.capacity <= 0 {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	v, exists := c.items[key]
	if !exists {
		return
	}

	c.ll.MoveToBack(v)
	return v.Value.(entry[K, V]).value, true
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}

	c.mutex.Lock()
	defer c.mutex.Unlock()
	v, exists := c.items[key]
	if exists {
		v.Value = entry[K, V]{key: key, value: value}
		return
	}

	if c.capacity == c.ll.Len() {
		toDelete := c.ll.Front()
		c.ll.Remove(toDelete)
		e := toDelete.Value.(entry[K, V])
		delete(c.items, e.key)
	}

	el := c.ll.PushBack(entry[K, V]{key: key, value: value})
	c.items[key] = el

	return
}

type LRU[K comparable, V any] interface {
	Get(K) (V, bool)
	Set(K, V)
}
