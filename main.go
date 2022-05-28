package main

import (
	"github.com/tidwall/gjson"
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func validateCurrencyCode(CurrencyCode string) bool {
	// description: validates currency code
	result := false
	// todo: work up a real test
	if CurrencyCode == "is in list of currencies" {
		result = true
	} else {
		result = false
	}
	return result
}

func getRate(CurrencyFrom string, CurrencyTo string) float64 {
	httpposturl := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/chart/%v%v%v", CurrencyFrom, CurrencyTo, "=x?corsDomain=finance.yahoo.com&range=1d&interval=1d")
	// todo: get rid of bytes.NewBuffer yet set a request header
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
	if response.StatusCode != 200 {
		fmt.Println("something was wrong. status code was not 200")
		os.Exit(response.StatusCode)
	}
	responseData, _ := ioutil.ReadAll(response.Body)
	// digs into the json and pulls out the value we want
	// todo: use std library to pull this value using structs
	value := gjson.GetBytes(responseData, "chart.result.0.meta.regularMarketPrice")
	regularMarketPrice, _ := strconv.ParseFloat(value.String(), 64)
	return regularMarketPrice

}

func main() {
	start := time.Now()
	Qty, _ := 1.0, 0
	lenArgs := len(os.Args)
	if lenArgs == 4 {
		Qty, _ = strconv.ParseFloat(os.Args[3], 64)
	}

	if lenArgs < 3 {
		fmt.Println("USAGE: con [currency_code] [currency_code] int")
		fmt.Println("EXAMPLE: con usd czk 100")
		fmt.Println("list of currencies: cat ~/git/scripts/currencies")
		os.Exit(2)
	}

	CurrencyFrom := strings.ToUpper(os.Args[1])
	CurrencyTo := strings.ToUpper(os.Args[2])
	CurrencyPair := fmt.Sprintf("%v%v", CurrencyFrom, CurrencyTo)
	CurrencyPairInverse := fmt.Sprintf("%v%v", CurrencyTo, CurrencyFrom)

	// attempt to get key 'CurrentPair' from redis
	result, err := redisGet(CurrencyPair)

	// if redis key does not exist, we get an error
	// and getRate then stick value in redis
	if err != nil {
		// get rate from API
		regularMarketPrice := getRate(CurrencyFrom, CurrencyTo)
		regularMarketPriceString := strconv.FormatFloat(regularMarketPrice, 'g', -1, 64)

		// convert rate to string
		regularMarketPriceInverseString := strconv.FormatFloat(1/regularMarketPrice, 'g', -1, 64)

		// store key-value pairs in redis
		redisSet(CurrencyPair, regularMarketPriceString)
		redisSet(CurrencyPairInverse, regularMarketPriceInverseString)

		// print rates
		printRates(regularMarketPrice, Qty, CurrencyFrom, CurrencyTo)
		printTally(regularMarketPrice, float64(Qty), CurrencyTo)
	} else {
		resultFloat64, _ := strconv.ParseFloat(result, 64)
		printRates(resultFloat64, Qty, CurrencyFrom, CurrencyTo)
		printTally(resultFloat64, float64(Qty), CurrencyTo)
	}
	fmt.Printf("\n\ntime taken: %v ms\n", time.Since(start).Milliseconds())

}

func printTally(regularMarketPrice float64, Qty float64, CurrencyTo string) {
	fmt.Printf("\n\n  %.2f %v\n\n", regularMarketPrice*float64(Qty), CurrencyTo)

}

func printRates(regularMarketPrice float64, Qty float64, CurrencyFrom string, CurrencyTo string) {
	fmt.Printf("\namount: %v %v\n\n", Qty, CurrencyFrom)
	fmt.Printf("1 %v = %v %v\n", CurrencyFrom, regularMarketPrice, CurrencyTo)
	fmt.Printf("1 %v = %.3f %v\n", CurrencyTo, 1/regularMarketPrice, CurrencyFrom)
}
