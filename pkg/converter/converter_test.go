// FILE: pkg/converter/converter_test.go
package converter_test

import (
	"context"
	"fmt"
	"testing"

	"currency_converter/pkg/converter"
)

// fakeFetcher implements converter.RateFetcher and tracks FetchRate calls.
type fakeFetcher struct {
	rate      float64
	calls     int
	shouldErr bool
}

func (f *fakeFetcher) FetchRate(ctx context.Context, from, to string) (float64, error) {
	f.calls++
	if f.shouldErr {
		return 0, fmt.Errorf("fetch error")
	}
	return f.rate, nil
}

// TestConvert_CachesRate ensures that Converter.Convert caches fetched rates
// so that subsequent calls for the same currency pair do not invoke the fetcher again.
func TestConvert_CachesRate(t *testing.T) {
	ff := &fakeFetcher{rate: 2.5}
	c := converter.New(ff)

	// First conversion should call the fetcher exactly once.
	rate1, inv1, result1, err := c.Convert(context.Background(), "USD", "EUR", 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rate1 != 2.5 {
		t.Errorf("expected rate 2.5, got %v", rate1)
	}
	if inv1 != 1/2.5 {
		t.Errorf("expected inverse rate %.6f, got %.6f", 1/2.5, inv1)
	}
	if result1 != 2.5*10 {
		t.Errorf("expected result %.2f, got %.2f", 2.5*10, result1)
	}
	if ff.calls != 1 {
		t.Errorf("expected fetcher to be called once, got %d", ff.calls)
	}

	// Second conversion for the same pair should use cache and not call fetcher again.
	rate2, inv2, result2, err := c.Convert(context.Background(), "USD", "EUR", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ff.calls != 1 {
		t.Errorf("expected fetcher to still be called once, got %d", ff.calls)
	}
	if rate2 != rate1 || inv2 != inv1 || result2 != 2.5*5 {
		t.Errorf("unexpected conversion values on cached call")
	}
}

// TestConvert_ErrorOnZeroRate ensures that Converter.Convert returns an error
// if the fetched rate is zero or negative.
func TestConvert_ErrorOnZeroRate(t *testing.T) {
	ff := &fakeFetcher{rate: 0}
	c := converter.New(ff)
	_, _, _, err := c.Convert(context.Background(), "USD", "GBP", 1)
	if err == nil {
		t.Fatal("expected error for zero rate, got nil")
	}
}
