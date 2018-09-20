package zizou

import (
	"fmt"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	DefaultKey = "w4erf3w4ref43t24rwgthg43t2r3fg"
	DefaultVal = `3w43eryduh23orregfw4r34f3rfwq34e53`

	NoExpiration   time.Duration = 0
	DTevictionTime time.Duration = 1000 * time.Millisecond
)

func timeGC() time.Duration {
	start := time.Now()
	runtime.GC()
	return time.Since(start)
}

func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
	fmt.Println()
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func intToStr(i int) string {
	return strconv.Itoa(i)
}

func TestCache(t *testing.T) {
	cnf := &Config{
		SweepTime: 0,
		ShardSize: 256,
	}
	tc, err := New(cnf)
	if err != nil {
		t.Error(err)
		return
	}

	tc.Set("a", 1, NoExpiration)
	tc.Set("b", "b", NoExpiration)
	tc.Set("c", 3.5, NoExpiration)

	x, found := tc.Get("a")
	if !found {
		t.Error("a was not found while getting a2")
	}
	if x == nil {
		t.Error("x for a is nil")
	} else if a2 := x.(int); a2+2 != 3 {
		t.Error("a2 (which should be 1) plus 2 does not equal 3; value:", a2)
	}

	x, found = tc.Get("b")
	if !found {
		t.Error("b was not found while getting b2")
	}
	if x == nil {
		t.Error("x for b is nil")
	} else if b2 := x.(string); b2+"B" != "bB" {
		t.Error("b2 (which should be b) plus B does not equal bB; value:", b2)
	}

	x, found = tc.Get("c")
	if !found {
		t.Error("c was not found while getting c2")
	}
	if x == nil {
		t.Error("x for c is nil")
	} else if c2 := x.(float64); c2+1.2 != 4.7 {
		t.Error("c2 (which should be 3.5) plus 1.2 does not equal 4.7; value:", c2)
	}
}

func TestCacheTimes(t *testing.T) {
	var found bool

	cnf := &Config{
		SweepTime: 0,
		ShardSize: 256,
	}
	tc, err := New(cnf)
	if err != nil {
		t.Error(err)
		return
	}

	tc.Set("a", 1, 2*time.Millisecond)
	tc.Set("b", 2, NoExpiration)
	tc.Set("c", 3, 20*time.Millisecond)
	tc.Set("d", 4, 70*time.Millisecond)

	<-time.After(25 * time.Millisecond)
	_, found = tc.Get("c")
	if found {
		t.Error("Found c when it should have been automatically deleted")
	}

	<-time.After(30 * time.Millisecond)
	_, found = tc.Get("a")
	if found {
		t.Error("Found a when it should have been automatically deleted")
	}

	_, found = tc.Get("b")
	if !found {
		t.Error("Did not find b even though it was set to never expire")
	}

	_, found = tc.Get("d")
	if !found {
		t.Error("Did not find d even though it was set to expire later than the default")
	}

	<-time.After(20 * time.Millisecond)
	_, found = tc.Get("d")
	if found {
		t.Error("Found d when it should have been automatically deleted (later than the default)")
	}
}

func BenchmarkCacheGetExpiring(b *testing.B) {
	benchmarkCacheGet(b, 50*time.Millisecond)
}

func BenchmarkCacheGetNotExpiring(b *testing.B) {
	benchmarkCacheGet(b, NoExpiration)
}

func benchmarkCacheGet(b *testing.B, exp time.Duration) {
	b.StopTimer()
	cnf := &Config{
		SweepTime: DTevictionTime,
		ShardSize: 256,
	}
	tc, err := New(cnf)
	if err != nil {
		b.Error(err)
		return
	}

	tc.Set("foo", DefaultVal, exp)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Get("foo")
	}
}

func BenchmarkCacheGetConcurrentExpiring(b *testing.B) {
	benchmarkCacheGetConcurrent(b, 50*time.Millisecond)
}

func BenchmarkCacheGetConcurrentNotExpiring(b *testing.B) {
	benchmarkCacheGetConcurrent(b, NoExpiration)
}

func benchmarkCacheGetConcurrent(b *testing.B, exp time.Duration) {
	b.StopTimer()
	cnf := &Config{
		SweepTime: DTevictionTime,
		ShardSize: 256,
	}
	tc, err := New(cnf)
	if err != nil {
		b.Error(err)
		return
	}

	tc.Set("foo", DefaultVal, exp)
	wg := new(sync.WaitGroup)
	workers := runtime.NumCPU()
	each := b.N / workers
	wg.Add(workers)
	b.StartTimer()
	for i := 0; i < workers; i++ {
		go func() {
			for j := 0; j < each; j++ {
				tc.Get("foo")
			}
			wg.Done()
		}()
	}
	wg.Wait()
}

func BenchmarkCacheSetExpiring(b *testing.B) {
	benchmarkCacheSet(b, 50*time.Millisecond)
}

func BenchmarkCacheSetNotExpiring(b *testing.B) {
	benchmarkCacheSet(b, NoExpiration)
}

func benchmarkCacheSet(b *testing.B, exp time.Duration) {
	b.StopTimer()
	cnf := &Config{
		SweepTime: exp,
		ShardSize: 256,
	}
	tc, err := New(cnf)
	if err != nil {
		b.Error(err)
		return
	}

	b.StartTimer()
	fmt.Println("len: ", b.N)
	for i := 0; i < b.N; i++ {
		tc.Set(intToStr(i), DefaultVal, exp)
	}
	PrintMemUsage()
	runtime.GC()
	fmt.Println("GC took: ", timeGC())
}

func BenchmarkCacheSetDelete(b *testing.B) {
	b.StopTimer()
	cnf := &Config{
		SweepTime: DTevictionTime,
		ShardSize: 256,
	}
	tc, err := New(cnf)
	if err != nil {
		b.Error(err)
		return
	}

	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", DefaultVal, NoExpiration)
		tc.Delete("foo")
	}
}

func BenchmarkCacheFlush(b *testing.B) {
	b.StopTimer()
	cnf := &Config{
		SweepTime: DTevictionTime,
		ShardSize: 256,
	}
	tc, err := New(cnf)
	if err != nil {
		b.Error(err)
		return
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", DefaultVal, NoExpiration)
	}
	tc.Flush()
}

func BenchmarkCacheMultipleSetFlush(b *testing.B) {
	b.StopTimer()
	cnf := &Config{
		SweepTime: DTevictionTime,
		ShardSize: 256,
	}
	tc, err := New(cnf)
	if err != nil {
		b.Error(err)
		return
	}
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set(intToStr(i), DefaultVal, NoExpiration)
	}
	tc.Flush()
}
