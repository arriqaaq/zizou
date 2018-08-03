# zizou
In memory cache implementation

Ideas:
- Make a circular shard ring, cache objects in individual shard for better concurrency and to avoid locks on the whole cache map object
- Use eviction policies of key using a list object, evict those at front
- Use ticker to clear off cache in diff shards
- Implement operations using cmdable interface as done in go-redis(design pattern)
- Check how GC can be avoided 
