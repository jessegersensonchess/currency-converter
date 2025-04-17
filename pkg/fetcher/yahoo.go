// pkg/fetcher/yahoo.go
package fetcher

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// YahooFetcher implements RateFetcher using two Yahoo endpoints with retry logic.
type YahooFetcher struct{}

// NewYahooFetcher constructs a YahooFetcher.
func NewYahooFetcher() *YahooFetcher {
	return &YahooFetcher{}
}

const (
	api1        = "https://query1.finance.yahoo.com/v7/finance/chart"
	api2        = "https://query2.finance.yahoo.com/v7/finance/chart"
	requestTO   = 2 * time.Second
	baseBackoff = 500 * time.Millisecond
	maxRetries  = 2
)

type yahooResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
}

// FetchRate tries api1 then api2. On 429 or network errors, it will retry up to maxRetries
// with exponential backoff. Other HTTP errors immediately fail over to the next endpoint.
func (f *YahooFetcher) FetchRate(ctx context.Context, from, to string) (float64, error) {
	endpoints := []string{api1, api2}

	for _, baseURL := range endpoints {
		backoff := baseBackoff

		for attempt := 1; attempt <= maxRetries; attempt++ {
			// bound each HTTP call to its own timeout
			reqCtx, cancel := context.WithTimeout(ctx, requestTO)
			defer cancel()

			url := fmt.Sprintf("%s/%s%s=x?range=1d&interval=1d", baseURL, from, to)
			req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
			if err != nil {
				log.Printf("YahooFetcher: build request error: %v", err)
				// retry building the request
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			// Set headers including a custom User-Agent to mimic a modern Firefox browser.
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:112.0) Gecko/20100101 Firefox/112.0")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				log.Printf("YahooFetcher: HTTP error (attempt %d): %v", attempt, err)
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				log.Printf("YahooFetcher: read body error: %v", err)
				return 0, err
			}

			if resp.StatusCode == http.StatusOK {
				var wrapper yahooResponse
				if err := json.Unmarshal(body, &wrapper); err != nil {
					log.Printf("YahooFetcher: JSON unmarshal error: %v", err)
					return 0, err
				}
				if len(wrapper.Chart.Result) == 0 {
					return 0, errors.New("YahooFetcher: no result in chart")
				}
				return wrapper.Chart.Result[0].Meta.RegularMarketPrice, nil
			}

			// on 429, retry; on other codes, break to next endpoint
			if resp.StatusCode == http.StatusTooManyRequests {
				log.Printf("YahooFetcher: 429 Too Many Requests (attempt %d), backing off %s", attempt, backoff)
				time.Sleep(backoff)
				backoff *= 2
				continue
			}

			log.Printf("YahooFetcher: non-200 %d: %s", resp.StatusCode, string(body))
			break
		}
		// if we exhausted retries on this endpoint, move on
	}

	return 0, fmt.Errorf("YahooFetcher: all endpoints failed for %s→%s", from, to)
}
