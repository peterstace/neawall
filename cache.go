package main

import (
	"sync"

	"golang.org/x/sync/singleflight"
)

type SingleFlightForeverCache[V any] struct {
	sf    singleflight.Group
	mu    sync.Mutex
	cache map[string]V
}

func NewSingleFlightForeverCache[V any]() *SingleFlightForeverCache[V] {
	return &SingleFlightForeverCache[V]{
		cache: make(map[string]V),
	}
}

func (c *SingleFlightForeverCache[V]) Get(k string, fallback func() (V, error)) (V, error) {
	c.mu.Lock()
	v, ok := c.cache[k]
	if ok {
		c.mu.Unlock()
		return v, nil
	}
	c.mu.Unlock()

	vAny, err, _ := c.sf.Do(k, func() (any, error) {
		return fallback()
	})
	if err != nil {
		var zero V
		return zero, err
	}
	v = vAny.(V)

	c.mu.Lock()
	c.cache[k] = v
	c.mu.Unlock()
	return v, nil
}
