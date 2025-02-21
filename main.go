// ISO_4217 currency converter
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"
)

// Global cache for storing conversion rates
var rateCache = make(map[string]float64)
var cacheMutex = &sync.Mutex{}

var (
	ctx     = context.Background()
	apiUrl1 = "https://query1.finance.yahoo.com/v7/finance/chart"
	apiUrl2 = "https://query2.finance.yahoo.com/v7/finance/chart"
)

type apiResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				Currency           string  `json:"currency"`
				Symbol             string  `json:"symbol"`
				RegularMarketTime  int     `json:"regularMarketTime"`
				RegularMarketPrice float64 `json:"regularMarketPrice"`
			} `json:"meta"`
			Timestamp []int `json:"timestamp"`
		} `json:"result"`
		Error interface{} `json:"error"`
	} `json:"chart"`
}

func readFile(path string) ([]byte, error) {
	parentPath, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	pullPath := filepath.Join(parentPath, path)
	file, err := os.Open(pullPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	return read(file)
}

func read(fd_r io.Reader) ([]byte, error) {
	br := bufio.NewReader(fd_r)
	var buf bytes.Buffer
	for {
		ba, isPrefix, err := br.ReadLine()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		buf.Write(ba)
		if !isPrefix {
			buf.WriteByte('\n')
		}
	}
	return buf.Bytes(), nil
}

func printList(path string) string {
	ba, err := readFile(path)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
	}
	return fmt.Sprintf("The content of '%s' : \n%s\n", path, ba)
}

// getData fetches JSON data from the given URL using the provided context.
// It now sets a custom User-Agent header and logs additional debugging info for non-200 responses.
func getData(ctx context.Context, url string) (apiResponse, error) {
	var c apiResponse
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return c, err
	}
	// Set headers including a custom User-Agent to mimic a modern Firefox browser.
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:112.0) Gecko/20100101 Firefox/112.0")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return c, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return c, err
	}

	if res.StatusCode != http.StatusOK {
		snippet := string(body)
		if len(snippet) > 512 {
			snippet = snippet[:512] + "..."
		}
		log.Printf("DEBUG: Non-200 response from URL %s: status code %d, body snippet: %s", url, res.StatusCode, snippet)
		return c, fmt.Errorf("non-200 HTTP status code: %d", res.StatusCode)
	}

	err = json.Unmarshal(body, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}

// getRate now tries apiUrl1 first and, if it fails (e.g. 429), waits briefly then tries apiUrl2.
func getRate(CurrencyFrom string, CurrencyTo string) (regularMarketPrice float64) {
	// Create a context with a 2-second timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	fetchFromURL := func(apiUrl, apiSource string) (float64, error) {
		url := fmt.Sprintf("%v/%v%v=x?range=1d&interval=1d", apiUrl, CurrencyFrom, CurrencyTo)
		apiResp, err := getData(ctx, url)
		if err != nil {
			log.Printf("Error fetching rate from %s (%s): %v", apiUrl, apiSource, err)
			return 0, err
		}
		for _, res := range apiResp.Chart.Result {
			log.Printf("Result received from %s (%s)", apiSource, apiUrl)
			return res.Meta.RegularMarketPrice, nil
		}
		return 0, fmt.Errorf("no result in response from %s", apiUrl)
	}

	// Try first endpoint.
	rate, err := fetchFromURL(apiUrl1, "apiUrl1")
	if err == nil && rate != 0 {
		return rate
	}

	// Wait briefly before trying the fallback endpoint.
	time.Sleep(300 * time.Millisecond)
	rate, err = fetchFromURL(apiUrl2, "apiUrl2")
	if err == nil && rate != 0 {
		return rate
	}

	// If both fail, return 0 (caller should handle this as an error condition).
	return 0
}

// CurrencyRequest represents a request for currency conversion.
type CurrencyRequest struct {
	CurrencyFrom string `json:"currency_from"`
	CurrencyTo   string `json:"currency_to"`
	Quantity     int    `json:"quantity"`
}

// CurrencyResponse represents the JSON response for a currency conversion.
type CurrencyResponse struct {
	Result   float64 `json:"result"`
	From     string  `json:"from"`
	To       string  `json:"to"`
	FromRate float64 `json:"from_rate"`
	ToRate   float64 `json:"to_rate"`
	Quantity int     `json:"quantity"`
}

func convertHandler(w http.ResponseWriter, r *http.Request) {
	var req CurrencyRequest

	// Decode the JSON body.
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error decoding JSON: %v", err), http.StatusBadRequest)
		return
	}

	CurrencyPair := req.CurrencyFrom + req.CurrencyTo
	CurrencyPairInverse := req.CurrencyTo + req.CurrencyFrom

	// Check the cache first.
	cacheMutex.Lock()
	rate, found := rateCache[CurrencyPair]
	cacheMutex.Unlock()

	if !found {
		// If not found in cache, get rate from API.
		rate = getRate(req.CurrencyFrom, req.CurrencyTo)
		if rate == 0 {
			http.Error(w, "Failed to retrieve rate", http.StatusInternalServerError)
			return
		}

		// Store rates in cache.
		cacheMutex.Lock()
		rateCache[CurrencyPair] = rate
		rateCache[CurrencyPairInverse] = 1 / rate
		cacheMutex.Unlock()
	}

	// Perform conversion.
	result := rate * float64(req.Quantity)

	// Determine the output format.
	format := r.URL.Query().Get("format")
	if format == "text" {
		inverseRate := 0.0
		if rate != 0 {
			inverseRate = 1 / rate
		}
		output := fmt.Sprintf("\nAmount: %d %s\n\n", req.Quantity, req.CurrencyFrom)
		output += fmt.Sprintf("1 %s = %.4f %s\n", req.CurrencyFrom, rate, req.CurrencyTo)
		output += fmt.Sprintf("1 %s = %.4f %s\n\n", req.CurrencyTo, inverseRate, req.CurrencyFrom)
		output += fmt.Sprintf("Result: %.2f %s\n", result, req.CurrencyTo)
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, output)
	} else {
		inverseRate := 0.0
		if rate != 0 {
			inverseRate = 1 / rate
		}
		resp := CurrencyResponse{
			Result:   result,
			From:     req.CurrencyFrom,
			To:       req.CurrencyTo,
			FromRate: rate,
			ToRate:   inverseRate,
			Quantity: req.Quantity,
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
		}
	}
}

func main() {
	var port string

	// Define a command-line flag for the port.
	flagPort := flag.String("p", "", "Port to run the currency converter server on")
	flag.Parse()

	// Check if the port is provided via a flag or environment variable.
	if *flagPort != "" {
		port = *flagPort
	} else if envPort, exists := os.LookupEnv("CURRENCY_CONVERTER_PORT"); exists {
		port = envPort
	} else {
		// Set to default port if not provided.
		port = "18880"
	}

	// Validate port number.
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("Invalid port number: %s", port)
	}

	// Set up HTTP route and handler.
	http.HandleFunc("/convert", convertHandler)

	// Start the HTTP server.
	fmt.Printf("Starting server at port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
