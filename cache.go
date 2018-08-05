package zizou

import (
	"time"
)

func New(sweepTime time.Duration) *Cache {
	nc := &Cache{
		shards:    make([]*shard, 256),
		hash:      newXXHash(),
		shardMask: 255,
	}
	for i := uint64(0); i < 256; i++ {
		nc.shards[i] = newShardWithSweeper(sweepTime)
	}
	return nc
}

type Cache struct {
	shards    []*shard
	hash      hasher
	shardMask uint64
}

func (c *Cache) getShard(hashedKey uint64) (shard *shard) {
	return c.shards[hashedKey&c.shardMask]
}

func (c *Cache) Get(k string) (interface{}, bool) {
	hashedKey := c.hash.Sum64(k)
	shard := c.getShard(hashedKey)
	return shard.Get(k)
}

func (c *Cache) Set(k string, v interface{}, dur time.Duration) error {
	hashedKey := c.hash.Sum64(k)
	shard := c.getShard(hashedKey)
	return shard.Set(k, v, dur)
}

func (c *Cache) Delete(k string) bool {
	hashedKey := c.hash.Sum64(k)
	shard := c.getShard(hashedKey)
	return shard.Delete(k)

}

func (c *Cache) Flush() {
	for _, shard := range c.shards {
		shard.Flush()
	}
}
