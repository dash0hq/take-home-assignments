package main

import (
	"sync"
	"testing"
	"time"
)

// TestSeriesCache_FirstSeenAlwaysWrites verifies a MetadataID never seen before triggers a write.
func TestSeriesCache_FirstSeenAlwaysWrites(t *testing.T) {
	c := newSeriesCache(10 * time.Minute)
	now := time.Now()

	if !c.shouldWrite(123, now) {
		t.Fatal("expected shouldWrite to return true for a MetadataID never seen before")
	}
}

// TestSeriesCache_SecondCallWithinTTLSkips verifies a repeat within the TTL skips the write.
func TestSeriesCache_SecondCallWithinTTLSkips(t *testing.T) {
	c := newSeriesCache(10 * time.Minute)
	now := time.Now()

	if !c.shouldWrite(123, now) {
		t.Fatal("expected first call to return true")
	}
	if c.shouldWrite(123, now.Add(1*time.Minute)) {
		t.Fatal("expected second call within TTL to return false (should skip metadata write)")
	}
}

// TestSeriesCache_ExpiresAfterTTL verifies a call past TTL expiry triggers a write again.
func TestSeriesCache_ExpiresAfterTTL(t *testing.T) {
	c := newSeriesCache(10 * time.Minute)
	now := time.Now()

	if !c.shouldWrite(123, now) {
		t.Fatal("expected first call to return true")
	}
	// Just past the TTL boundary.
	if !c.shouldWrite(123, now.Add(10*time.Minute+time.Second)) {
		t.Fatal("expected call past TTL expiry to return true again")
	}
}

// TestSeriesCache_IndependentIDsDoNotInterfere verifies caching one MetadataID doesn't affect another's state.
func TestSeriesCache_IndependentIDsDoNotInterfere(t *testing.T) {
	c := newSeriesCache(10 * time.Minute)
	now := time.Now()

	if !c.shouldWrite(1, now) {
		t.Fatal("expected shouldWrite(1) to return true")
	}
	if !c.shouldWrite(2, now) {
		t.Fatal("expected shouldWrite(2) to return true even though a different ID was just cached")
	}
	if c.shouldWrite(1, now) {
		t.Fatal("expected shouldWrite(1) to return false on immediate repeat")
	}
}

// TestSeriesCache_ConcurrentAccess exercises the sharded locking under concurrent use; run with -race.
func TestSeriesCache_ConcurrentAccess(t *testing.T) {
	c := newSeriesCache(10 * time.Minute)
	now := time.Now()

	var wg sync.WaitGroup
	const goroutines = 50
	const idsPerGoroutine = 200

	for g := 0; g < goroutines; g++ {
		wg.Add(1)
		go func(offset int) {
			defer wg.Done()
			for i := 0; i < idsPerGoroutine; i++ {
				id := uint64(offset*idsPerGoroutine + i)
				c.shouldWrite(id, now)
			}
		}(g)
	}
	wg.Wait()
	// No assertions beyond "doesn't race or panic" - correctness of
	// hit/miss behavior under single-threaded access is covered above.
}
