package zizou

import (
	"runtime"
	"sync"
	"time"
)

const (
	DefaultEvictionTime = 600 * time.Second
)

type evictor interface {
	Run(*cache)
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

func (s *sweeper) Run(c *cache) {
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

func stopSweeper(c *Cache) {
	c.sweeper.Stop()
}

func runSweeper(c *cache, dur time.Duration) {
	swp := newSweeper(dur)
	c.sweeper = swp
	go swp.Run(c)
}

type item struct {
	value    interface{}
	expireAt int64
}

func New(sweepTime time.Duration) *Cache {
	nc := &Cache{newCache()}
	if sweepTime > 0 {
		// This trick ensures that the sweeper goroutine (which--granted it
		// was enabled--is running DeleteExpired on c forever) does not keep
		// the returned C object from being garbage collected. When it is
		// garbage collected, the finalizer stops the sweeper goroutine, after
		// which c can be collected.
		// Source: https://github.com/patrickmn/go-cache/blob/master/cache.go#L1093:6
		runSweeper(nc.cache, sweepTime)
		runtime.SetFinalizer(nc, stopSweeper)
	}
	return nc
}

type Cache struct {
	*cache
}

func newCache() *cache {
	return &cache{
		data: make(map[string]item),
	}
}

type cache struct {
	// map of string items to avoid gc, GC kicks in on every item of map
	// if it has pointers as key/val pairs
	data map[string]item
	mu   sync.RWMutex

	sweeper evictor
}

func (c *cache) set(k string, v interface{}, dur time.Duration) error {
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

func (c *cache) Set(k string, v interface{}, dur time.Duration) error {
	c.mu.Lock()
	c.set(k, v, dur)
	c.mu.Unlock()
	return nil
}

func (c *cache) get(k string) (item, bool) {
	item, found := c.data[k]
	return item, found
}

/*
	If key has expired on a get, it's deleted in the same call
*/
func (c *cache) Get(k string) (interface{}, bool) {
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

func (c *cache) TTL(k string) time.Time {
	val, found := c.get(k)
	if found {
		return time.Unix(0, val.expireAt)
	}
	return time.Time{}
}

func (c *cache) delete(k string) bool {
	_, found := c.data[k]
	if found {
		delete(c.data, k)
		return true
	}
	return false
}

func (c *cache) Delete(k string) bool {
	c.mu.Lock()
	found := c.delete(k)
	c.mu.Unlock()
	return found
}

// This is going to be painfully slow as the rwlock will be on the entire map
// during the period of evicting expired keys. Will be resolved in future using
// sharded ring buffers
func (c *cache) Evict() {
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

func (c *cache) Flush() {
	c.mu.Lock()
	c.data = map[string]item{}
	c.mu.Unlock()
}
