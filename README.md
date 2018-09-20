# zizou
In memory cache implementation with high concurrency

![zizou](https://i.pinimg.com/originals/32/14/45/321445a239a44f8efeec3a6cf04a6d13.jpg)




## Features

* Store millions of entries
* High concurrent thread-safe access
* Expiration support
* Shard support to avoid locks on whole db during any concurrent read/writes/deletes

## Example Usage

```go
config:=zizou.Config{
	SweepTime:10*time.Minute,
	ShardSize: 256,
}
cache,err:=zizou.New(&config)
cache.Set("foo", "bar", 10*time.Second)
item, found := cache.Get("foo")
```



## Performance

```bash
		BenchmarkCacheGetExpiring-4                	20000000	        70.0 ns/op
		BenchmarkCacheGetNotExpiring-4             	20000000	       119 ns/op
		BenchmarkCacheGetConcurrentExpiring-4      	20000000	        59.3 ns/op
		BenchmarkCacheGetConcurrentNotExpiring-4   	20000000	        91.3 ns/op
		BenchmarkCacheSetExpiring-4                	10000000	       172 ns/op
		BenchmarkCacheSetNotExpiring-4             	10000000	       134 ns/op
		BenchmarkCacheSetDelete-4                  	 5000000	       343 ns/op
```


## TODO

- Store data as a map of string and byte array to avoid lookups on object, which will be copied in memory
- Make a circular shard ring, cache objects in individual shard for better concurrency and to avoid locks on the whole cache map object (LRU implementation)
- Use eviction policies of key using a list object, evict those at front
- Implement operations using cmdable interface as done in go-redis(design pattern)
- Check how GC can be avoided 

Zizou Image source: https://www.pinterest.ca/pin/436075176391410672/?lp=true
