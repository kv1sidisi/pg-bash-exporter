package cache

import (
	"sync"
	"time"
)

type Cache struct {
	mu    sync.Mutex
	items map[string]Item
}

type Item struct {
	Value      string
	Err        error
	Expiration time.Time
}

func New() *Cache {
	c := &Cache{
		items: make(map[string]Item),
	}

	go c.runGBCollector(10 * time.Minute)
	return c
}

func (c *Cache) runGBCollector(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		for key, item := range c.items {
			if !item.Expiration.IsZero() && time.Now().After(item.Expiration) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

func (c *Cache) Set(key, value string, err error, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var expiration time.Time
	if ttl > 0 {
		expiration = time.Now().Add(ttl)
	}

	c.items[key] = Item{
		Value:      value,
		Err:        err,
		Expiration: expiration,
	}
}

func (c *Cache) Get(key string) (string, error, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	item, found := c.items[key]
	if !found {
		return "", nil, false
	}

	if !item.Expiration.IsZero() && time.Now().After(item.Expiration) {
		delete(c.items, key)
		return "", nil, false
	}

	return item.Value, item.Err, true
}
