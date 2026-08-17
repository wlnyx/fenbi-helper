package web

import (
	"sync"
	"time"
)

// memCache 简单内存缓存（TTL 过期）。
type memCache struct {
	mu    sync.Mutex
	items map[string]cacheItem
	ttl   time.Duration
}

type cacheItem struct {
	value  interface{}
	expiry time.Time
}

func newMemCache(ttl time.Duration) *memCache {
	return &memCache{items: map[string]cacheItem{}, ttl: ttl}
}

func (c *memCache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[key]
	if !ok {
		return nil, false
	}
	if time.Now().After(it.expiry) {
		delete(c.items, key)
		return nil, false
	}
	return it.value, true
}

func (c *memCache) Set(key string, value interface{}) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL 带独立 TTL 写入缓存项。
func (c *memCache) SetWithTTL(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items[key] = cacheItem{value: value, expiry: time.Now().Add(ttl)}
}

func (c *memCache) Invalidate(prefix string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for k := range c.items {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(c.items, k)
		}
	}
}
