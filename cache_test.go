package zizou

import (
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	NoExpiration  time.Duration = 0
	DEvictionTime               = 1 * time.Second
	DefaultKey                  = "w4erf3w4ref43t24rwgthg43t2r3fg"
	DefaultVal                  = `curl -XPOST http://130.211.114.240:8000/alexander/token -d'{"debug": true,"game_id": "00000100","user": {"advid": "string","ai5": "testai5","ua": "string","ip": "106.51.65.205","optout": 0,"consent": 0},"app": {"ver": "string","num": 0,"bundle": "string","engine": "string"},"sdk": {"num": 0,"ver": "8.8.3","mock-admob": {"ver": "1.0"},"units": {"float": ["float-0103","float-0104"],"native": ["unit-0101","unit-0102","unit-0102"]}},"device": {"os": {"pltfrm": "string","ver": "string","num": "0","apilvl": 0},"maker": "string","model": "string","scrn": {"h": 0,"w": 0,"d": 0,"di": 0},"locale": "string"},"carrier": {"name": "string","ct": 0,"cr": "string","hni": 0}}'
{"token":"96a94af5-60e5-4ccc-76af-2c98132b1dd0","next":{"partners":[{"name":"mock-admob","prio":21,"ecpm":0.05,"cmp_id":"7001","conf":{"app_id":"ca-app-pub-4073866383873410~7558583524","plcmnt_id":"ca-app-pub-3940256099942544/2247696110"}}]}}`
)

func intToStr(i int) string {
	return strconv.Itoa(i)
}

func TestCache(t *testing.T) {
	tc := New(0)

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

	tc := New(10 * time.Millisecond)
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
	tc := New(NoExpiration)
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
	tc := New(NoExpiration)
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
	tc := New(DEvictionTime)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", DefaultVal, exp)
	}
}

func BenchmarkCacheSetDelete(b *testing.B) {
	b.StopTimer()
	tc := New(NoExpiration)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", DefaultVal, NoExpiration)
		tc.Delete("foo")
	}
}

func BenchmarkCacheFlush(b *testing.B) {
	b.StopTimer()
	tc := New(NoExpiration)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set("foo", DefaultVal, NoExpiration)
	}
	tc.Flush()
}

func BenchmarkCacheMultipleSetFlush(b *testing.B) {
	b.StopTimer()
	tc := New(NoExpiration)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		tc.Set(intToStr(i), DefaultVal, NoExpiration)
	}
	tc.Flush()
}
