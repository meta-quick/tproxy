package main

import (
	"sync"
	"time"
)

type Closeable interface {
	Close() error
}

type CacheItem[T any] struct {
	Value      T
	Expiration time.Time
}

// Bucket 结构体定义
type Bucket[T any] struct {
	items map[string]CacheItem[T]
	mu    sync.RWMutex
}

// TimeCache 泛型缓存实现
type TimeCache[T any] struct {
	buckets []Bucket[T]
	ttl     time.Duration
	ticker  *time.Ticker
	stopCh  chan struct{}
}

// NewTimeCache 创建一个新的 TimeCache 实例
func NewTimeCache[T any](ttl time.Duration, bucketCount int) *TimeCache[T] {
	cache := &TimeCache[T]{
		buckets: make([]Bucket[T], bucketCount),
		ttl:     ttl,
		ticker:  time.NewTicker(ttl / 2),
		stopCh:  make(chan struct{}),
	}

	for i := range cache.buckets {
		cache.buckets[i] = Bucket[T]{items: make(map[string]CacheItem[T])}
	}

	go cache.startEviction()
	return cache
}

// getBucket 根据键获取桶
func (c *TimeCache[T]) getBucket(key string) *Bucket[T] {
	// 简单的哈希函数选择桶
	bucketIndex := hash(key) % len(c.buckets)
	return &c.buckets[bucketIndex]
}

// Set 设置缓存项
func (c *TimeCache[T]) Set(key string, value T) {
	bucket := c.getBucket(key)
	bucket.mu.Lock()
	defer bucket.mu.Unlock()
	bucket.items[key] = CacheItem[T]{
		Value:      value,
		Expiration: time.Now().Add(c.ttl),
	}
}

// Get 获取缓存项
func (c *TimeCache[T]) Get(key string) (T, bool) {
	bucket := c.getBucket(key)
	bucket.mu.RLock()
	defer bucket.mu.RUnlock()
	item, found := bucket.items[key]
	if !found || time.Now().After(item.Expiration) {
		var zero T
		return zero, false
	}

	// 更新过期时间
	item.Expiration = time.Now().Add(c.ttl)
	bucket.items[key] = item

	return item.Value, true
}

// startEviction 定期清理过期项
func (c *TimeCache[T]) startEviction() {
	for {
		select {
		case <-c.ticker.C:
			c.evictExpired()
		case <-c.stopCh:
			c.ticker.Stop()
			return
		}
	}
}

// evictExpired 从每个桶中删除过期项
func (c *TimeCache[T]) evictExpired() {
	for i := range c.buckets {
		bucket := &c.buckets[i]
		bucket.mu.Lock()
		for key, item := range bucket.items {
			if time.Now().After(item.Expiration) {
				delete(bucket.items, key)
				var intf interface{} = item.Value
				if closeable, ok := intf.(Closeable); ok {
					_ = closeable.Close()
				}
			}
		}
		bucket.mu.Unlock()
	}
}

// Stop 停止定时器和清理操作
func (c *TimeCache[T]) Stop() {
	close(c.stopCh)
}

// hash 简单的哈希函数
func hash(key string) int {
	h := 0
	for i := 0; i < len(key); i++ {
		h = 31*h + int(key[i])
	}
	return h
}
