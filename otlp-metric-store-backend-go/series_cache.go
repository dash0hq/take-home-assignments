package main

import (
	"sync"
	"time"
)

// shardCount must be a power of two so shard selection can use a fast
// bitmask instead of modulo.
const shardCount = 16

// seriesCache tracks which MetadataIDs had their metadata row written
// recently, so the ingest path can skip redundant metadata inserts. Sharded
// to avoid lock contention; ReplacingMergeTree is the correctness backstop,
// this is purely a throughput optimization.
type seriesCache struct {
	shards [shardCount]cacheShard
	ttl    time.Duration
}

type cacheShard struct {
	mu   sync.Mutex
	seen map[uint64]time.Time
}

// newSeriesCache creates a cache whose entries expire after ttl, keeping memory bounded.
func newSeriesCache(ttl time.Duration) *seriesCache {
	c := &seriesCache{ttl: ttl}
	for i := range c.shards {
		c.shards[i].seen = make(map[uint64]time.Time)
	}
	return c
}

// shardFor returns the shard responsible for the given MetadataID.
func (c *seriesCache) shardFor(id uint64) *cacheShard {
	return &c.shards[id&(shardCount-1)]
}

// shouldWrite reports whether metadata for id needs (re-)writing, marking it seen if so.
func (c *seriesCache) shouldWrite(id uint64, now time.Time) bool {
	shard := c.shardFor(id)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	if last, ok := shard.seen[id]; ok && now.Sub(last) < c.ttl {
		return false
	}
	shard.seen[id] = now
	return true
}
