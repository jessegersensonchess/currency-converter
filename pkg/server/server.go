// pkg/server/server.go
package server

import (
	"encoding/json"
	"fmt"
	"net/http"

	"currency_converter/pkg/converter"
)

// NewHandler returns an http.Handler for /convert.
func NewHandler(cv *converter.Converter) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		type reqBody struct {
			From string  `json:"currency_from"`
			To   string  `json:"currency_to"`
			Qty  float64 `json:"quantity"`
		}
		var body reqBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		rate, invRate, result, err := cv.Convert(r.Context(), body.From, body.To, body.Qty)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if r.URL.Query().Get("format") == "text" {
			text := formatTextResponse(body.Qty, body.From, body.To, rate, invRate, result)
			w.Header().Set("Content-Type", "text/plain")
			w.Write([]byte(text))
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
		resp := jsonResp{result, body.From, body.To, rate, invRate, body.Qty}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
}

func formatTextResponse(qty float64, from, to string, rate, invRate, result float64) string {
	return fmt.Sprintf(
		"1 %s = %.4f %s\n1 %s = %.4f %s\n\n%.2f %s\n",
		from, rate, to,
		to, invRate, from,
		result, to,
	)
}
