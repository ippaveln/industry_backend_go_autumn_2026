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
	return &LRUCache[K, V]{
		capacity: capacity,
		items:    make(map[K]*list.Element),
	}
}

func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	elem, ok := c.items[key]
	if !ok {
		return value, false
	}

	// Успешное чтение делает элемент самым свежим.
	c.ll.MoveToBack(elem)

	item := elem.Value.(entry[K, V])
	return item.value, true
}

func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}

	// Если ключ уже существует, обновляем только значение.
	// Порядок давности не меняем.
	if elem, ok := c.items[key]; ok {
		elem.Value = entry[K, V]{
			key:   key,
			value: value,
		}
		return
	}

	// Если кэш заполнен, удаляем самый старый элемент.
	if c.ll.Len() >= c.capacity {
		oldest := c.ll.Front()
		item := oldest.Value.(entry[K, V])

		delete(c.items, item.key)
		c.ll.Remove(oldest)
	}

	// Добавляем новый элемент в конец списка.
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
