// pkg/server/server.go
package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"currency_converter/pkg/converter"
)

// NewHandler returns an http.Handler for /convert/{from}/{to}/{qty}.
func NewHandler(cv *converter.Converter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		from, to, qty, err := parseConvertPath(r.URL.Path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		rate, invRate, result, err := cv.Convert(r.Context(), from, to, qty)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if r.URL.Query().Get("format") == "text" {
			text := formatTextResponse(qty, from, to, rate, invRate, result)
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			_, _ = w.Write([]byte(text))
			return
		}

		type jsonResp struct {
			Result   float64 `json:"result"`
			From     string  `json:"from"`
			To       string  `json:"to"`
			FromRate float64 `json:"from_rate"`
			ToRate   float64 `json:"to_rate"`
			Quantity float64 `json:"quantity"`
		}

		resp := jsonResp{
			Result:   result,
			From:     from,
			To:       to,
			FromRate: rate,
			ToRate:   invRate,
			Quantity: qty,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	})
}

func parseConvertPath(path string) (from, to string, qty float64, err error) {
	const prefix = "/convert/"

	if !strings.HasPrefix(path, prefix) {
		return "", "", 0, fmt.Errorf("invalid path")
	}

	parts := strings.Split(strings.TrimPrefix(path, prefix), "/")
	if len(parts) != 3 {
		return "", "", 0, fmt.Errorf("expected /convert/{from}/{to}/{qty}")
	}

	from = strings.ToUpper(strings.TrimSpace(parts[0]))
	to = strings.ToUpper(strings.TrimSpace(parts[1]))
	if from == "" || to == "" {
		return "", "", 0, fmt.Errorf("currency codes must not be empty")
	}

	qty, err = strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid quantity")
	}

	return from, to, qty, nil
}

func formatTextResponse(qty float64, from, to string, rate, invRate, result float64) string {
	return fmt.Sprintf(
		"1 %s = %.4f %s\n1 %s = %.4f %s\n\n%.2f %s\n",
		from, rate, to,
		to, invRate, from,
		result, to,
	)
}
