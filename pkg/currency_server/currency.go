package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

// Chart struct is retained for potential future use.
type Chart struct {
	Result []struct {
		Meta struct {
			Currency             string  `json:"currency"`
			Symbol               string  `json:"symbol"`
			ExchangeName         string  `json:"exchangeName"`
			InstrumentType       string  `json:"instrumentType"`
			FirstTradeDate       int     `json:"firstTradeDate"`
			RegularMarketTime    int     `json:"regularMarketTime"`
			Gmtoffset            int     `json:"gmtoffset"`
			Timezone             string  `json:"timezone"`
			ExchangeTimezoneName string  `json:"exchangeTimezoneName"`
			RegularMarketPrice   float64 `json:"regularMarketPrice"`
			ChartPreviousClose   float64 `json:"chartPreviousClose"`
			PriceHint            int     `json:"priceHint"`
			CurrentTradingPeriod struct {
				Pre struct {
					Timezone  string `json:"timezone"`
					End       int    `json:"end"`
					Start     int    `json:"start"`
					Gmtoffset int    `json:"gmtoffset"`
				} `json:"pre"`
				Regular struct {
					Timezone  string `json:"timezone"`
					End       int    `json:"end"`
					Start     int    `json:"start"`
					Gmtoffset int    `json:"gmtoffset"`
				} `json:"regular"`
				Post struct {
					Timezone  string `json:"timezone"`
					End       int    `json:"end"`
					Start     int    `json:"start"`
					Gmtoffset int    `json:"gmtoffset"`
				} `json:"post"`
			} `json:"currentTradingPeriod"`
			DataGranularity string   `json:"dataGranularity"`
			Range           string   `json:"range"`
			ValidRanges     []string `json:"validRanges"`
		} `json:"meta"`
		Timestamp  int `json:"timestamp"`
		Indicators struct {
			Quote []struct {
				Open   []float64 `json:"open"`
				High   []float64 `json:"high"`
				Low    []float64 `json:"low"`
				Close  []float64 `json:"close"`
				Volume []int     `json:"volume"`
			} `json:"quote"`
			Adjclose []struct {
				Adjclose []float64 `json:"adjclose"`
			} `json:"adjclose"`
		} `json:"indicators"`
	} `json:"result"`
	Error interface{} `json:"error"`
}

func validateCurrencyCode(currencyCode string) bool {
	// TODO: Implement a proper validation for currency codes.
	return currencyCode == "is in list of currencies"
}

func getRate(currencyFrom, currencyTo string) float64 {
	// Build URL for the API call.
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/chart/%v%v=x?corsDomain=finance.yahoo.com&range=1d&interval=1d", currencyFrom, currencyTo)

	// Create a new GET request with no body.
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		panic(err)
	}

	// Set headers, including a modern Firefox User-Agent.
	request.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:112.0) Gecko/20100101 Firefox/112.0")
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		fmt.Printf("Error: received status code %d\n", response.StatusCode)
		io.Copy(os.Stdout, response.Body)
		os.Exit(response.StatusCode)
	}

	responseData, err := io.ReadAll(response.Body)
	if err != nil {
		panic(err)
	}

	// Extract the regularMarketPrice using gjson.
	value := gjson.GetBytes(responseData, "chart.result.0.meta.regularMarketPrice")
	regularMarketPrice, err := strconv.ParseFloat(value.String(), 64)
	if err != nil {
		panic(err)
	}
	return regularMarketPrice
}

func printRates(regularMarketPrice, qty float64, currencyFrom, currencyTo string) {
	fmt.Printf("Amount: %v %v\n", qty, currencyFrom)
	fmt.Printf("1 %v = %v %v\n", currencyFrom, regularMarketPrice, currencyTo)
	fmt.Printf("1 %v = %.3f %v\n", currencyTo, 1/regularMarketPrice, currencyFrom)
}

func printTally(regularMarketPrice, qty float64, currencyTo string) {
	fmt.Printf("\n\n%.2f %v\n\n", regularMarketPrice*qty, currencyTo)
}

func main() {
	start := time.Now()
	qty := 1.0
	args := os.Args

	if len(args) < 3 {
		fmt.Println("USAGE: con [currency_code] [currency_code] [amount]")
		fmt.Println("EXAMPLE: con USD CZK 100")
		fmt.Println("List of currencies: cat ~/git/scripts/currencies")
		os.Exit(2)
	}

	currencyFrom := strings.ToUpper(args[1])
	currencyTo := strings.ToUpper(args[2])
	if len(args) == 4 {
		var err error
		qty, err = strconv.ParseFloat(args[3], 64)
		if err != nil {
			fmt.Printf("Invalid quantity: %v\n", args[3])
			os.Exit(1)
		}
	}

	regularMarketPrice := getRate(currencyFrom, currencyTo)
	printRates(regularMarketPrice, qty, currencyFrom, currencyTo)
	fmt.Printf("\nTime taken: %v ms", time.Since(start).Milliseconds())
	printTally(regularMarketPrice, qty, currencyTo)
}
