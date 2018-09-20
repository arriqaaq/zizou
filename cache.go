package zizou

import (
	"errors"
	"time"
)

var (
	ERR_INVALID_CONFIG = errors.New("invalid configuration")
)

func isPowerOfTwo(num uint64) bool {
	return (num != 0) && ((num & (num - 1)) == 0)
}

type Config struct {
	SweepTime time.Duration
	ShardSize uint64
}

func (c *Config) Validate() bool {
	if c.SweepTime < 0 {
		return false
	}
	if !isPowerOfTwo(c.ShardSize) {
		return false
	}
	return true
}

func New(cnf *Config) (*Cache, error) {
	isValid := cnf.Validate()
	if !isValid {
		return nil, ERR_INVALID_CONFIG
	}

	nc := &Cache{
		shards:    make([]*shard, cnf.ShardSize),
		hash:      newXXHash(),
		shardMask: cnf.ShardSize - 1,
	}
	for i := uint64(0); i < cnf.ShardSize; i++ {
		nc.shards[i] = newShardWithSweeper(cnf.SweepTime)
	}
	return nc, nil
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
