package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const endpoint = "http://127.0.0.1:31880/convert"

func main() {
	transport := &http.Transport{
		Proxy: nil,

		MaxIdleConns:        1,
		MaxIdleConnsPerHost: 1,
		MaxConnsPerHost:     1,
		IdleConnTimeout:     20 * time.Minute,
		DisableCompression:  true,

		DialContext: (&net.Dialer{
			Timeout:   2 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,

		ForceAttemptHTTP2:     false,
		TLSHandshakeTimeout:   2 * time.Second,
		ResponseHeaderTimeout: 2 * time.Second,
		ExpectContinueTimeout: 0,
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   5 * time.Second,
	}

	if err := warmUp(client); err != nil {
		fmt.Fprintf(os.Stderr, "warm-up failed: %v\n", err)
	} else {
		fmt.Println("warm-up complete")
	}

	fmt.Println("Enter: FROM TO [QTY]")
	fmt.Println("Examples:")
	fmt.Println("  usd eur")
	fmt.Println("  usd eur 12.5")
	fmt.Println("Type 'quit' or 'exit' to stop.")

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			fmt.Println()
			break
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if line == "quit" || line == "exit" {
			break
		}

		parts := strings.Fields(line)
		if len(parts) < 2 || len(parts) > 3 {
			fmt.Println("usage: FROM TO [QTY]")
			continue
		}

		from := strings.ToUpper(parts[0])
		to := strings.ToUpper(parts[1])

		qty := 1.0
		if len(parts) == 3 {
			v, err := strconv.ParseFloat(parts[2], 64)
			if err != nil {
				fmt.Printf("bad quantity: %v\n", err)
				continue
			}
			qty = v
		}

		isVerbose := true
		if err := doConvert(client, from, to, qty, isVerbose); err != nil {
			fmt.Printf("request failed: %v\n", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "input error: %v\n", err)
	}
}

func warmUp(client *http.Client) error {
	return doConvert(client, "USD", "EUR", 1, false)
}

func doConvert(client *http.Client, from, to string, qty float64, verbose bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	qtyStr := strconv.FormatFloat(qty, 'f', -1, 64)
	reqURL := fmt.Sprintf(
		"%s/%s/%s/%s?format=text",
		endpoint,
		url.PathEscape(from),
		url.PathEscape(to),
		url.PathEscape(qtyStr),
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Close = false

	requestStart := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(requestStart)

	if err != nil {
		if verbose {
			fmt.Printf("elapsed=%v\n", elapsed)
		}
		return fmt.Errorf("perform request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)

	if err != nil {
		if verbose {
			fmt.Printf("elapsed=%v\n", elapsed)
		}
		return fmt.Errorf("read response: %w", err)
	}

	if verbose {
		fmt.Printf("%s\n", respBody)
		fmt.Printf("elapsed=%v\n", elapsed)
	}

	return nil
}
