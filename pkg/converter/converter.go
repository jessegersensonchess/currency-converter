// pkg/converter/converter.go
package converter

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// RateFetcher defines anything that can fetch a currency pair rate.
type RateFetcher interface {
	FetchRate(ctx context.Context, from, to string) (float64, error)
}

// RateCacheItem holds the exchange rate and the timestamp when it was cached.
type RateCacheItem struct {
	Rate      float64
	Timestamp time.Time
}

// Converter holds a fetcher and a thread‑safe cache.
type Converter struct {
	fetcher RateFetcher
	cache   map[string]RateCacheItem
	mu      sync.Mutex
	ttl     time.Duration // TTL for cache items
}

// New constructs a Converter.
func New(f RateFetcher) *Converter {
	return &Converter{
		fetcher: f,
		cache:   make(map[string]RateCacheItem),
		ttl:     43200 * time.Second, // TTL is 7200 seconds (2 hours)
	}
}

// Convert returns (rate, inverseRate, convertedAmount, error).
func (c *Converter) Convert(ctx context.Context, from, to string, qty float64) (
	rate, inverse, result float64, err error,
) {
	key := from + to
	invKey := to + from

	// Check cache and TTL
	c.mu.Lock()
	cachedItem, ok := c.cache[key]
	c.mu.Unlock()

	// If rate is in cache and TTL is not expired, return cached rate
	if ok && time.Since(cachedItem.Timestamp) < c.ttl {
		rate = cachedItem.Rate
	} else {
		// If not in cache or TTL expired, fetch a new rate
		rate, err = c.fetcher.FetchRate(ctx, from, to)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("fetch rate: %w", err)
		}
		if rate <= 0 {
			return 0, 0, 0, fmt.Errorf("invalid rate %.6f for %s→%s", rate, from, to)
		}

		// Update cache with new rate and current timestamp
		c.mu.Lock()
		c.cache[key] = RateCacheItem{Rate: rate, Timestamp: time.Now()}
		c.cache[invKey] = RateCacheItem{Rate: 1 / rate, Timestamp: time.Now()}
		c.mu.Unlock()
	}

	// Calculate inverse and result
	inverse = 1 / rate
	result = rate * qty
	return rate, inverse, result, nil
}
