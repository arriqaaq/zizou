package zizou

import (
	hash1 "github.com/cespare/xxhash"
	"math/rand"
	"runtime"
	"sync"
	"time"
)

const (
	DefaultEvictionTime = 600 * time.Second
	MinimumStartupTime  = 300 * time.Millisecond
	MaximumStartupTime  = 2 * MinimumStartupTime
)

// Used to put a random delay before start of each shard, so as to not
// let various shards lock at the same time
func startupDelay() time.Duration {
	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	d, delta := MinimumStartupTime, (MaximumStartupTime - MinimumStartupTime)
	if delta > 0 {
		d += time.Duration(rand.Int63n(int64(delta)))
	}
	return d
}

type evictor interface {
	Run(*shard)
	Stop()
}

func newSweeper(interval time.Duration) evictor {
	return &sweeper{
		interval: interval,
		stopC:    make(chan bool),
	}
}

type sweeper struct {
	interval time.Duration
	stopC    chan bool
}

func (s *sweeper) Run(c *shard) {
	<-time.After(startupDelay())
	ticker := time.NewTicker(s.interval)
	for {
		select {
		case <-ticker.C:
			c.Evict()
		case <-s.stopC:
			ticker.Stop()
			return
		}
	}
}

func (s *sweeper) Stop() {
	s.stopC <- true
}

func stopSweeper(c *shard) {
	c.sweeper.Stop()
}

func runSweeper(c *shard, dur time.Duration) {
	swp := newSweeper(dur)
	c.sweeper = swp
	go swp.Run(c)
}

type item struct {
	value    interface{}
	expireAt int64
}

type hasher interface {
	Sum64(string) uint64
}

func newXXHash() hasher {
	return xxHash{}
}

// https://cyan4973.github.io/xxHash/
type xxHash struct {
}

func (x xxHash) Sum64(data string) uint64 {
	return hash1.Sum64([]byte(data))
}

func newShard() *shard {
	return &shard{
		data: make(map[string]item),
	}
}

func newShardWithSweeper(sweepTime time.Duration) *shard {
	nc := &shard{
		data: make(map[string]item),
	}
	if sweepTime > 0 {
		// This trick ensures that the sweeper goroutine (which--granted it
		// was enabled--is running DeleteExpired on c forever) does not keep
		// the returned C object from being garbage collected. When it is
		// garbage collected, the finalizer stops the sweeper goroutine, after
		// which c can be collected.
		// Source: https://github.com/patrickmn/go-cache/blob/master/cache.go#L1093:6
		runSweeper(nc, sweepTime)
		runtime.SetFinalizer(nc, stopSweeper)
	}
	return nc
}

type shard struct {
	// map of string items to avoid gc, GC kicks in on every item of map
	// if it has pointers as key/val pairs
	data map[string]item
	mu   sync.RWMutex

	sweeper evictor
}

func (c *shard) set(k string, v interface{}, dur time.Duration) error {
	var d int64
	if dur > 0 {
		d = time.Now().Add(dur).UnixNano()
	}
	c.data[k] = item{
		value:    v,
		expireAt: d,
	}
	return nil
}

func (c *shard) Set(k string, v interface{}, dur time.Duration) error {
	c.mu.Lock()
	c.set(k, v, dur)
	c.mu.Unlock()
	return nil
}

func (c *shard) get(k string) (item, bool) {
	item, found := c.data[k]
	return item, found
}

/*
	If key has expired on a get, it's deleted in the same call
*/
func (c *shard) Get(k string) (interface{}, bool) {
	c.mu.RLock()
	val, found := c.get(k)
	c.mu.RUnlock()
	if found {
		now := time.Now().UnixNano()
		if val.expireAt > 0 && now > val.expireAt {
			c.mu.Lock()
			c.delete(k)
			c.mu.Unlock()
			return nil, false
		}
		return val.value, true
	}
	return nil, false
}

func (c *shard) TTL(k string) time.Time {
	val, found := c.get(k)
	if found {
		return time.Unix(0, val.expireAt)
	}
	return time.Time{}
}

func (c *shard) delete(k string) bool {
	_, found := c.data[k]
	if found {
		delete(c.data, k)
		return true
	}
	return false
}

func (c *shard) Delete(k string) bool {
	c.mu.Lock()
	found := c.delete(k)
	c.mu.Unlock()
	return found
}

// This is going to be painfully slow as the rwlock will be on the entire map
// during the period of evicting expired keys.
func (c *shard) Evict() {
	now := time.Now().UnixNano()
	delKeys := make([]string, 0, 1000)
	c.mu.RLock()
	for key, val := range c.data {
		if val.expireAt > 0 && now > val.expireAt {
			delKeys = append(delKeys, key)
		}
	}
	c.mu.RUnlock()

	// Could be made into a diff go routine in future
	c.mu.Lock()
	for _, key := range delKeys {
		c.delete(key)
	}
	c.mu.Unlock()
}

func (c *shard) Flush() {
	c.mu.Lock()
	c.data = map[string]item{}
	c.mu.Unlock()
}
