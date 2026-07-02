// Copyright 2019 Ipregistry (https://ipregistry.co).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ipregistry

import (
	"container/list"
	"sync"
	"time"
)

// Cache abstracts the storage used by a Client to memoize IP lookups.
// Implementations must be safe for concurrent use by multiple goroutines.
//
// Only successful single and batch IP lookups are cached. Origin lookups are
// never cached, because the requester IP is only known from the response.
type Cache interface {
	// Get returns the cached value for key, and whether it was present.
	Get(key string) (*IPInfo, bool)
	// Set stores value under key.
	Set(key string, value *IPInfo)
	// Invalidate removes the entry for key, if present.
	Invalidate(key string)
	// InvalidateAll removes every entry.
	InvalidateAll()
}

// noopCache is the default Cache; it stores nothing. It is used when no cache is
// configured, so lookups always hit the API and data is never stale.
type noopCache struct{}

func (noopCache) Get(string) (*IPInfo, bool) { return nil, false }
func (noopCache) Set(string, *IPInfo)        {}
func (noopCache) Invalidate(string)          {}
func (noopCache) InvalidateAll()             {}

// Default in-memory cache settings.
const (
	defaultCacheMaxSize = 4096
	defaultCacheTTL     = 10 * time.Minute
)

// InMemoryCache is a thread-safe, in-process Cache with time-based expiration
// and a bounded size using least-recently-used eviction. The zero value is not
// usable; construct one with NewInMemoryCache.
type InMemoryCache struct {
	mu      sync.Mutex
	maxSize int
	ttl     time.Duration
	ll      *list.List               // front = most recently used
	items   map[string]*list.Element // key -> element
	now     func() time.Time         // overridable clock, for testing
}

// cacheEntry is the value stored in each list element.
type cacheEntry struct {
	key       string
	value     *IPInfo
	expiresAt time.Time
}

// CacheOption customizes an InMemoryCache.
type CacheOption func(*InMemoryCache)

// WithMaxSize sets the maximum number of entries the cache holds before it
// starts evicting the least recently used entry. A value <= 0 leaves the
// default (4096).
func WithMaxSize(n int) CacheOption {
	return func(c *InMemoryCache) {
		if n > 0 {
			c.maxSize = n
		}
	}
}

// WithTTL sets how long an entry stays valid after being written. A value <= 0
// leaves the default (10 minutes).
func WithTTL(d time.Duration) CacheOption {
	return func(c *InMemoryCache) {
		if d > 0 {
			c.ttl = d
		}
	}
}

// NewInMemoryCache creates an InMemoryCache. Without options it holds up to 4096
// entries for 10 minutes each.
func NewInMemoryCache(opts ...CacheOption) *InMemoryCache {
	c := &InMemoryCache{
		maxSize: defaultCacheMaxSize,
		ttl:     defaultCacheTTL,
		ll:      list.New(),
		items:   make(map[string]*list.Element),
		now:     time.Now,
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// Get returns the cached value for key when present and unexpired.
func (c *InMemoryCache) Get(key string) (*IPInfo, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	entry := el.Value.(*cacheEntry)
	if c.now().After(entry.expiresAt) {
		c.removeElement(el)
		return nil, false
	}
	c.ll.MoveToFront(el)
	return entry.value, true
}

// Set stores value under key, refreshing its expiration and evicting the least
// recently used entry if the cache is full.
func (c *InMemoryCache) Set(key string, value *IPInfo) {
	c.mu.Lock()
	defer c.mu.Unlock()

	expiresAt := c.now().Add(c.ttl)
	if el, ok := c.items[key]; ok {
		entry := el.Value.(*cacheEntry)
		entry.value = value
		entry.expiresAt = expiresAt
		c.ll.MoveToFront(el)
		return
	}

	el := c.ll.PushFront(&cacheEntry{key: key, value: value, expiresAt: expiresAt})
	c.items[key] = el

	if c.ll.Len() > c.maxSize {
		if oldest := c.ll.Back(); oldest != nil {
			c.removeElement(oldest)
		}
	}
}

// Invalidate removes the entry for key, if present.
func (c *InMemoryCache) Invalidate(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.removeElement(el)
	}
}

// InvalidateAll removes every entry.
func (c *InMemoryCache) InvalidateAll() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ll.Init()
	c.items = make(map[string]*list.Element)
}

// Len returns the current number of entries, including any not yet expired but
// possibly stale. It is primarily useful in tests.
func (c *InMemoryCache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// removeElement must be called with the mutex held.
func (c *InMemoryCache) removeElement(el *list.Element) {
	c.ll.Remove(el)
	delete(c.items, el.Value.(*cacheEntry).key)
}
