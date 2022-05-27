// Convert from one currency into another
package main

import (
	"github.com/tidwall/gjson"
	//	"encoding/json"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

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

func main() {
	start := time.Now()
	Qty, _ := 1.0, 0

	if len(os.Args) == 4 {
		Qty, _ = strconv.ParseFloat(os.Args[3], 64)
	}

	if len(os.Args) < 3 {
		fmt.Println("USAGE: con [currency_code] [currency_code] int")
		fmt.Println("EXAMPLE: con usd czk 100")
		//		fmt.Println("'CZD' is not a valid currency code. Did you mean")
		fmt.Println("list of currencies: cat ~/git/scripts/currencies")

		os.Exit(2)
	}

	CurrencyFrom := strings.ToUpper(os.Args[1])
	CurrencyTo := strings.ToUpper(os.Args[2])
	httpposturl := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/chart/%v%v%v", CurrencyFrom, CurrencyTo, "=x?corsDomain=finance.yahoo.com&range=1d&interval=1d")
	var jsonData = []byte(`{
		"b": "l"
	}`)

	request, error := http.NewRequest("GET", httpposturl, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		panic(error)
	}
	defer response.Body.Close()
	responseData, _ := ioutil.ReadAll(response.Body)
	value := gjson.GetBytes(responseData, "chart.result.0.meta.regularMarketPrice")
	regularMarketPrice, _ := strconv.ParseFloat(value.String(), 64)
	fmt.Println("amount:", Qty, CurrencyFrom, "\n")
	fmt.Println("1", CurrencyFrom, "=", regularMarketPrice, CurrencyTo)
	fmt.Printf("1 %v = %.3f %v\n", CurrencyTo, 1/regularMarketPrice, CurrencyFrom)
	fmt.Println("\ntime taken: ", time.Since(start))
	fmt.Printf("\n\n  %.2f %v\n\n", regularMarketPrice*float64(Qty), CurrencyTo)
}
