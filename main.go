package main

import (
	"github.com/tidwall/gjson"
	"github.com/go-redis/redis/v8"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"context"
	"time"
)

var ctx = context.Background()

// todo: use configuration management
// cache conversion rates in redis for 1 day
const redisTTL = 86400

func redisSet(key string, val string) {
	Rdb := redis.NewClient(&redis.Options{
		// todo: replace hardcoded values
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	// TTL is
	err := Rdb.Set(ctx, key, val, redisTTL*time.Second).Err()
	a := "1"
	if err != nil {
		//println("redis error:", err)
		a = ""
	}
	fmt.Sprintf("%v", a)
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


func validateCurrencyCode(CurrencyCode string) bool {
	// description: validates currency code
	// todo: work up a real test
	if CurrencyCode == "is in list of currencies" {
		return true
	}
	return false
}

func getRate(CurrencyFrom string, CurrencyTo string) float64 {
	url := fmt.Sprintf("https://query1.finance.yahoo.com/v7/finance/chart/%v%v%v", CurrencyFrom, CurrencyTo, "=x?range=1d&interval=1d")
	request, error := http.NewRequest("GET", url, nil)
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, error := client.Do(request)
	if error != nil {
		panic(error)
	}
	defer response.Body.Close()
	if response.StatusCode != 200 {
		fmt.Println("something was wrong. status code was not 200\n check https://en.wikipedia.org/wiki/ISO_4217 for valid currency codes")
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
	Qty := 1.0
	lenArgs := len(os.Args)
	if lenArgs == 4 {
		Qty, _ = strconv.ParseFloat(os.Args[3], 64)
	}

	if lenArgs < 3 {
		fmt.Println("USAGE: ./currency-converter [currency_code] [currency_code] int")
		fmt.Println("EXAMPLE: ./currency-converter usd eur 100")
		fmt.Println("list of currency codes: https://en.wikipedia.org/wiki/ISO_4217)")
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

}

func printTally(regularMarketPrice float64, Qty float64, CurrencyTo string) {
	fmt.Printf("\n\n  %.2f %v\n\n", regularMarketPrice*float64(Qty), CurrencyTo)

}

func printRates(regularMarketPrice float64, Qty float64, CurrencyFrom string, CurrencyTo string) {
	fmt.Printf("\namount: %v %v\n\n", Qty, CurrencyFrom)
	fmt.Printf("1 %v = %v %v\n", CurrencyFrom, regularMarketPrice, CurrencyTo)
	fmt.Printf("1 %v = %.3f %v\n", CurrencyTo, 1/regularMarketPrice, CurrencyFrom)
}
