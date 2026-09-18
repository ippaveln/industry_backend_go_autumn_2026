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
	panic("TODO: implement")
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	panic("TODO: implement")
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	panic("TODO: implement")
}
