package remap

import (
	"regexp"
	"sync"
)

type ReMap[T any] struct {
	kv map[*regexp.Regexp]T
	mu sync.RWMutex
}

func (r *ReMap[T]) Put(k *regexp.Regexp, v T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.kv[k] = v
}

func (r *ReMap[T]) Get(k *regexp.Regexp) (T, bool) {
	v, exists := r.kv[k]
	return v, exists
}

func (r *ReMap[T]) Find(k string) ([]T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	values := make([]T, 0, 8)
	found := false
	for storedKey, storedValue := range r.kv {
		if storedKey.MatchString(k) {
			values = append(values, storedValue)
			found = true
		}
	}
	return values, found
}

func (r *ReMap[T]) Entries() map[*regexp.Regexp]T {
	return r.kv
}

func (r *ReMap[T]) Delete(k *regexp.Regexp) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.kv, k)
}

func NewReMap[T any]() *ReMap[T] {
	return &ReMap[T]{kv: make(map[*regexp.Regexp]T)}
}
