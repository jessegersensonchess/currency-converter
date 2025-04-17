// pkg/converter/converter.go
package converter

import (
	"context"
	"fmt"
	"sync"
)

// RateFetcher defines anything that can fetch a currency pair rate.
type RateFetcher interface {
	FetchRate(ctx context.Context, from, to string) (float64, error)
}

// Converter holds a fetcher and a thread‑safe cache.
type Converter struct {
	fetcher RateFetcher
	cache   map[string]float64
	mu      sync.Mutex
}

// New constructs a Converter.
func New(f RateFetcher) *Converter {
	return &Converter{
		fetcher: f,
		cache:   make(map[string]float64),
	}
}

// Convert returns (rate, inverseRate, convertedAmount, error).
func (c *Converter) Convert(ctx context.Context, from, to string, qty int) (
	rate, inverse, result float64, err error,
) {
	key := from + to
	invKey := to + from

	// Check cache
	c.mu.Lock()
	rate, ok := c.cache[key]
	c.mu.Unlock()

	if !ok {
		rate, err = c.fetcher.FetchRate(ctx, from, to)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("fetch rate: %w", err)
		}
		if rate <= 0 {
			return 0, 0, 0, fmt.Errorf("invalid rate %.6f for %s→%s", rate, from, to)
		}
		c.mu.Lock()
		c.cache[key] = rate
		c.cache[invKey] = 1 / rate
		c.mu.Unlock()
	}

	inverse = 1 / rate
	result = rate * float64(qty)
	return rate, inverse, result, nil
}
