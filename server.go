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
	"strings"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
)

// Global cache for storing conversion rates
var rateCache = make(map[string]float64)
var cacheMutex = &sync.Mutex{}

var ctx = context.Background()

const (
	redisTTL = 86400
)

func redisSet(key string, val string) {
	Rdb := redis.NewClient(&redis.Options{
		// todo: replace hardcoded values
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	err := Rdb.Set(ctx, key, val, redisTTL*time.Second).Err()
	if err != nil {
	}
	return
}

func redisGet(key string) (string, error) {
	Rdb := redis.NewClient(&redis.Options{
		// todo: replace hardcoded values
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

	val, err := Rdb.Get(ctx, key).Result()
	if err == redis.Nil {
		fmt.Printf("key=%v does not exist in redis\n", key)
	} else if err != nil {
		//println("redis error:", err)
	} else {
		//fmt.Println(key, val)
	}
	return val, err
}

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
func getData(url string) (apiResponse, error) {
	c := apiResponse{}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return c, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return c, err
	}
	if res.StatusCode != 200 {
		fmt.Println("ERROR: HTTP response was not 200, exiting. status code was not 200\n check https://en.wikipedia.org/wiki/ISO_4217 for valid currency codes")
		os.Exit(res.StatusCode)
	}

	if res.Body != nil {
		defer res.Body.Close()
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

func getRate(CurrencyFrom string, CurrencyTo string) (regularMarketPrice float64) {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/chart/%v%v%v", CurrencyFrom, CurrencyTo, "=x?range=1d&interval=1d")
	apiResponse, err := getData(url)
	if err != nil {
		log.Fatal(err)
	}

	for _, i := range apiResponse.Chart.Result {
		regularMarketPrice = i.Meta.RegularMarketPrice
	}
	return
}

func convertCurrency(CurrencyFrom string, CurrencyTo string, Qty int) {
	CurrencyFrom = strings.ToUpper(CurrencyFrom)
	CurrencyTo = strings.ToUpper(CurrencyTo)
	CurrencyPair := CurrencyFrom + CurrencyTo
	CurrencyPairInverse := CurrencyTo + CurrencyFrom

	cacheMutex.Lock()
	rate, found := rateCache[CurrencyPair]
	cacheMutex.Unlock()

	if !found {
		// If not found in cache, get rate from API
		rate = getRate(CurrencyFrom, CurrencyTo)

		// Store rates in cache
		cacheMutex.Lock()
		rateCache[CurrencyPair] = rate
		rateCache[CurrencyPairInverse] = 1 / rate
		cacheMutex.Unlock()
	}

	// Convert and print rates
	result := rate * float64(Qty)
	printRates(rate, Qty, CurrencyFrom, CurrencyTo)
	printTally(result, Qty, CurrencyTo)
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

// CurrencyRequest represents a request for currency conversion
type CurrencyRequest struct {
	CurrencyFrom string `json:"currency_from"`
	CurrencyTo   string `json:"currency_to"`
	Quantity     int    `json:"quantity"`
}

// CurrencyResponse represents a response with conversion results
//type CurrencyResponse struct {
//	Result float64 `json:"result"`
//	From   string  `json:"from"`
//	To     string  `json:"to"`
//}

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

	// Perform conversion
	rate := getRate(req.CurrencyFrom, req.CurrencyTo)
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

func xxconvertHandler(w http.ResponseWriter, r *http.Request) {
	var req CurrencyRequest

	// Decode the JSON body
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Perform conversion
	rate := getRate(req.CurrencyFrom, req.CurrencyTo)
	result := rate * float64(req.Quantity)

	// Calculate inverse rate
	inverseRate := 0.0
	if rate != 0 {
		inverseRate = 1 / rate
	}

	// Create response with rates and quantity
	resp := CurrencyResponse{
		Result:   result,
		From:     req.CurrencyFrom,
		To:       req.CurrencyTo,
		FromRate: rate,
		ToRate:   inverseRate,
		Quantity: req.Quantity,
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func printTally(regularMarketPrice float64, Qty int, CurrencyTo string) {
	fmt.Printf("\n\n  %.2f %v\n\n", regularMarketPrice*float64(Qty), CurrencyTo)
}

func printRates(regularMarketPrice float64, Qty int, CurrencyFrom string, CurrencyTo string) {
	fmt.Printf("\namount: %v %v\n\n", Qty, CurrencyFrom)
	fmt.Printf("1 %v = %v %v\n", CurrencyFrom, regularMarketPrice, CurrencyTo)
	fmt.Printf("1 %v = %.3f %v\n", CurrencyTo, 1/regularMarketPrice, CurrencyFrom)
}
