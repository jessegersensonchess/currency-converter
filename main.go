// ISO_4217 currency converter
package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

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

	// attempt to get key 'CurrentPair' from redis
	result, err := redisGet(CurrencyPair)

	// if redisGet returns an err, "err" is not nil.
	// for example, if the key is not found, or redis is not running
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
		printTally(regularMarketPrice, Qty, CurrencyTo)
	} else {
		resultFloat64, _ := strconv.ParseFloat(result, 64)
		printRates(resultFloat64, Qty, CurrencyFrom, CurrencyTo)
		printTally(resultFloat64, Qty, CurrencyTo)
	}
}

func main() {
	lenArgs := len(os.Args)
	Qty := 1
	if lenArgs < 3 {
		fmt.Println("USAGE: ./currency-converter [currency_code] [currency_code] int")
		fmt.Println("EXAMPLE: ./currency-converter usd eur 100")
		fmt.Println("list of currency codes: https://en.wikipedia.org/wiki/ISO_4217)")
		fmt.Println(printList("list"))
		os.Exit(2)
	} else {
		if lenArgs >= 4 {
			Qty, _ = strconv.Atoi(os.Args[3])
		}
	}
	convertCurrency(strings.ToUpper(os.Args[1]), strings.ToUpper(os.Args[2]), Qty)
}

func printTally(regularMarketPrice float64, Qty int, CurrencyTo string) {
	fmt.Printf("\n\n  %.2f %v\n\n", regularMarketPrice*float64(Qty), CurrencyTo)
}

func printRates(regularMarketPrice float64, Qty int, CurrencyFrom string, CurrencyTo string) {
	fmt.Printf("\namount: %v %v\n\n", Qty, CurrencyFrom)
	fmt.Printf("1 %v = %v %v\n", CurrencyFrom, regularMarketPrice, CurrencyTo)
	fmt.Printf("1 %v = %.3f %v\n", CurrencyTo, 1/regularMarketPrice, CurrencyFrom)
}
