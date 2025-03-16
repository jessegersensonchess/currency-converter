package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// Test for the read() function using an io.Reader.
func TestRead(t *testing.T) {
	input := "line1\nline2\n"
	reader := bytes.NewBufferString(input)
	result, err := read(reader)
	if err != nil {
		t.Fatalf("read returned error: %v", err)
	}
	if string(result) != input {
		t.Errorf("Expected %q, got %q", input, result)
	}
}

// Test for printList() by writing a temporary file.
func TestPrintList(t *testing.T) {
	filename := "testfile.txt"
	content := "Hello, world!\nThis is a test."
	// Create the temporary file in the current working directory.
	if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temporary file: %v", err)
	}
	defer os.Remove(filename)

	output := printList(filename)
	if !strings.Contains(output, content) {
		t.Errorf("Output does not contain expected content. Got: %q", output)
	}
}

// Test getData() by using a dummy HTTP server.
func TestGetData(t *testing.T) {
	// Dummy JSON response with a valid conversion rate.
	responseJSON := `{
		"chart": {
			"result": [
				{
					"meta": {
						"currency": "USD",
						"symbol": "USDEUR=x",
						"regularMarketTime": 123456789,
						"regularMarketPrice": 1.23
					},
					"timestamp": [123456789]
				}
			],
			"error": null
		}
	}`

	// Create a test server that returns the dummy response.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseJSON))
	}))
	defer ts.Close()

	resp, err := getData(context.Background(), ts.URL)
	if err != nil {
		t.Fatalf("getData returned error: %v", err)
	}
	if len(resp.Chart.Result) == 0 {
		t.Fatalf("Expected at least one result")
	}
	rate := resp.Chart.Result[0].Meta.RegularMarketPrice
	if rate != 1.23 {
		t.Errorf("Expected rate 1.23, got %v", rate)
	}
}

// Test getRate() when the first API endpoint returns a valid result.
func TestGetRate_SuccessApiUrl1(t *testing.T) {
	// Clear the cache for a clean test.
	cacheMutex.Lock()
	rateCache = make(map[string]float64)
	cacheMutex.Unlock()

	// Create a test server for apiUrl1 that returns a valid rate.
	responseJSON := `{
		"chart": {
			"result": [
				{
					"meta": {
						"currency": "USD",
						"symbol": "USDEUR=x",
						"regularMarketTime": 123456789,
						"regularMarketPrice": 1.5
					},
					"timestamp": [123456789]
				}
			],
			"error": null
		}
	}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseJSON))
	}))
	defer ts.Close()

	// Override the global API URLs for testing.
	originalApiUrl1 := apiUrl1
	originalApiUrl2 := apiUrl2
	apiUrl1 = ts.URL
	apiUrl2 = ts.URL
	defer func() {
		apiUrl1 = originalApiUrl1
		apiUrl2 = originalApiUrl2
	}()

	rate := getRate("USD", "EUR")
	if rate != 1.5 {
		t.Errorf("Expected rate 1.5, got %v", rate)
	}
}

// Test getRate() by simulating an error on apiUrl1 and using the fallback apiUrl2.
func TestGetRate_FallbackToApiUrl2(t *testing.T) {
	// Clear the cache.
	cacheMutex.Lock()
	rateCache = make(map[string]float64)
	cacheMutex.Unlock()

	// Create a test server for apiUrl1 that returns an error.
	tsError := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}))
	defer tsError.Close()

	// Create a test server for apiUrl2 that returns a valid rate.
	responseJSON := `{
		"chart": {
			"result": [
				{
					"meta": {
						"currency": "USD",
						"symbol": "USDEUR=x",
						"regularMarketTime": 123456789,
						"regularMarketPrice": 2.0
					},
					"timestamp": [123456789]
				}
			],
			"error": null
		}
	}`
	tsValid := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(responseJSON))
	}))
	defer tsValid.Close()

	// Override global API URLs.
	originalApiUrl1 := apiUrl1
	originalApiUrl2 := apiUrl2
	apiUrl1 = tsError.URL
	apiUrl2 = tsValid.URL
	defer func() {
		apiUrl1 = originalApiUrl1
		apiUrl2 = originalApiUrl2
	}()

	rate := getRate("USD", "EUR")
	if rate != 2.0 {
		t.Errorf("Expected rate 2.0 from fallback, got %v", rate)
	}
}

// Test the HTTP handler for JSON output.
func TestConvertHandler_JSON(t *testing.T) {
	// Clear the cache and set known values.
	cacheMutex.Lock()
	rateCache = make(map[string]float64)
	rateCache["USDEUR"] = 0.8
	rateCache["EURUSD"] = 1.25
	cacheMutex.Unlock()

	reqBody := `{"currency_from": "USD", "currency_to": "EUR", "quantity": 10}`
	req := httptest.NewRequest("POST", "/convert", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	convertHandler(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %v", res.Status)
	}

	var resp CurrencyResponse
	if err := json.NewDecoder(res.Body).Decode(&resp); err != nil {
		t.Fatalf("Error decoding JSON response: %v", err)
	}

	expectedResult := 10 * 0.8
	if resp.Result != expectedResult {
		t.Errorf("Expected result %v, got %v", expectedResult, resp.Result)
	}
	if resp.From != "USD" || resp.To != "EUR" {
		t.Errorf("Unexpected currency fields in response")
	}
	if resp.FromRate != 0.8 || resp.ToRate != 1.25 {
		t.Errorf("Unexpected rate values in response")
	}
}

// Test the HTTP handler for text output.
func TestConvertHandler_Text(t *testing.T) {
	// Clear the cache and set known values.
	cacheMutex.Lock()
	rateCache = make(map[string]float64)
	rateCache["USDEUR"] = 1.2
	rateCache["EURUSD"] = 1 / 1.2 // approximately 0.8333
	cacheMutex.Unlock()

	reqBody := `{"currency_from": "USD", "currency_to": "EUR", "quantity": 5}`
	req := httptest.NewRequest("POST", "/convert?format=text", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	convertHandler(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK, got %v", res.Status)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("Error reading response body: %v", err)
	}
	bodyStr := string(body)
	if !strings.Contains(bodyStr, "Amount: 5 USD") {
		t.Errorf("Expected amount line in text response, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "1 USD =") || !strings.Contains(bodyStr, "EUR") {
		t.Errorf("Expected conversion rate line in text response, got: %s", bodyStr)
	}
	if !strings.Contains(bodyStr, "Result:") {
		t.Errorf("Expected result line in text response, got: %s", bodyStr)
	}
}

// Test the HTTP handler with invalid JSON input.
func TestConvertHandler_InvalidJSON(t *testing.T) {
	reqBody := `invalid json`
	req := httptest.NewRequest("POST", "/convert", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	convertHandler(w, req)

	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Errorf("Expected status Bad Request, got %v", res.Status)
	}
}
