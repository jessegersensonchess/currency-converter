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
	"io/ioutil"
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

var ctx = context.Background()
var apiUrl1 = "https://query1.finance.yahoo.com/v7/finance/chart"
var apiUrl2 = "https://query2.finance.yahoo.com/v7/finance/chart"

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

// GET url and return a struct (from https://codezup.com/fetch-parse-json-from-http-endpoint-golang/)
// getData now accepts a context for timeout control
func getData(ctx context.Context, url string) (apiResponse, error) {
	c := apiResponse{}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return c, err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return c, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return c, fmt.Errorf("non-200 HTTP status code: %d", res.StatusCode)
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return c, err
	}

	err = json.Unmarshal(body, &c)
	if err != nil {
		return c, err
	}
	return c, nil
}

// getRate is modified to include a timeout
func getRate(CurrencyFrom string, CurrencyTo string) (regularMarketPrice float64) {
	// Creating a context with a 2-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2000*time.Millisecond)
	defer cancel()

	// Channel to receive the first successful result
	resultChan := make(chan float64, 2) // Buffered to hold 2 results

	// Function to fetch rate and send result to channel
	fetchRate := func(apiUrl string, apiSource string) {
		url := fmt.Sprintf("%v/%v%v%v", apiUrl, CurrencyFrom, CurrencyTo, "=x?range=1d&interval=1d")
		apiResponse, err := getData(ctx, url) // Passing context to getData
		if err != nil {
			log.Println("Error fetching rate from:", apiUrl, "Error:", err)
			return
		}

		for _, i := range apiResponse.Chart.Result {
			log.Printf("Result received from %s", apiSource) // Logging the API source
			resultChan <- i.Meta.RegularMarketPrice
			break // Break after the first result to prevent multiple sends on channel
		}
	}

	// Start goroutines for each API URL
	go fetchRate(apiUrl1, "apiUrl1")
	go fetchRate(apiUrl2, "apiUrl2")

	// Use select to wait for the first result or timeout
	select {
	case regularMarketPrice = <-resultChan:
	case <-ctx.Done():
		log.Println("Request timed out")
	}

	return
}

// CurrencyRequest represents a request for currency conversion
type CurrencyRequest struct {
	CurrencyFrom string `json:"currency_from"`
	CurrencyTo   string `json:"currency_to"`
	Quantity     int    `json:"quantity"`
}

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

	// Decode the JSON body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	CurrencyPair := req.CurrencyFrom + req.CurrencyTo
	CurrencyPairInverse := req.CurrencyTo + req.CurrencyFrom

	// try adding cache
	cacheMutex.Lock()
	rate, found := rateCache[CurrencyPair]
	cacheMutex.Unlock()

	if !found {
		// If not found in cache, get rate from API
		rate = getRate(req.CurrencyFrom, req.CurrencyTo)

		// Store rates in cache
		cacheMutex.Lock()
		rateCache[CurrencyPair] = rate
		rateCache[CurrencyPairInverse] = 1 / rate
		cacheMutex.Unlock()
	}

	// Perform conversion
	//rate := getRate(req.CurrencyFrom, req.CurrencyTo)
	result := rate * float64(req.Quantity)

	// Determine the output format
	format := r.URL.Query().Get("format")

	if format == "text" {
		// Text format output
		inverseRate := 0.0
		if rate != 0 {
			inverseRate = 1 / rate
		}

		output := fmt.Sprintf("\namount: %d %s\n\n", req.Quantity, req.CurrencyFrom)
		output += fmt.Sprintf("1 %s = %.4f %s\n", req.CurrencyFrom, rate, req.CurrencyTo)
		output += fmt.Sprintf("1 %s = %.4f %s\n\n", req.CurrencyTo, inverseRate, req.CurrencyFrom)
		output += fmt.Sprintf("  %.2f %s\n", result, req.CurrencyTo)

		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprint(w, output)
	} else {
		// JSON format output
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
		json.NewEncoder(w).Encode(resp)
	}

}

func main() {
	var port string

	// Define a command-line flag
	flagPort := flag.String("p", "", "Port to run the currency converter server on")
	flag.Parse()

	// Check if the port is provided via a flag
	if *flagPort != "" {
		port = *flagPort
	} else {
		// Check if the port is provided via an environment variable
		envPort, exists := os.LookupEnv("CURRENCY_CONVERTER_PORT")
		if exists {
			port = envPort
		} else {
			// Set to default port if neither flag nor environment variable is set
			port = "18880"
		}
	}

	// Ensure port is a valid integer
	if _, err := strconv.Atoi(port); err != nil {
		log.Fatalf("Invalid port number: %s", port)
	}

	// HTTP route and handler setup
	http.HandleFunc("/convert", convertHandler)

	// Start the HTTP server
	fmt.Printf("Starting server at port %s...\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
