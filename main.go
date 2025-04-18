// cmd/currency/main.go
package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"currency_converter/pkg/converter"
	"currency_converter/pkg/fetcher"
	"currency_converter/pkg/server"
)

//go:embed VERSION_NUMBER
var files embed.FS

// read it at init time
var version = func() string {
	data, err := files.ReadFile("VERSION_NUMBER")
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(data))
}()

func main() {
	if len(os.Args) > 1 && os.Args[1] == "version" {
		fmt.Println(version)
		os.Exit(0)
	}

	// Flags
	cliMode := flag.Bool("cli", false, "Run a one-shot conversion on the CLI and exit")
	from := flag.String("from", "", "Source currency code (e.g. USD)")
	to := flag.String("to", "", "Target currency code (e.g. EUR)")
	qty := flag.Float64("qty", 1.0, "Amount to convert")

	portFlag := flag.String("p", "", "Port for HTTP server (default 18880)")
	flag.Parse()

	// Build a Converter with a YahooFetcher
	rateFetcher := fetcher.NewYahooFetcher()
	cv := converter.New(rateFetcher)

	if *cliMode {
		if *from == "" || *to == "" {
			log.Fatal("cli mode requires both -from and -to")
		}
		rate, invRate, result, err := cv.Convert(context.Background(), *from, *to, *qty)
		if err != nil {
			log.Fatalf("conversion failed: %v", err)
		}
		fmt.Printf("1 %s = %.4f %s\n", *from, rate, *to)
		fmt.Printf("1 %s = %.4f %s\n", *to, invRate, *from)
		fmt.Printf("\n%.2f %s\n", result, *to)
		return
	}

	// HTTP server
	port := defaultPort(*portFlag)
	handler := server.NewHandler(cv)
	http.Handle("/convert", handler)

	fmt.Printf("Starting HTTP server on :%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

func defaultPort(flagPort string) string {
	if flagPort != "" {
		return flagPort
	}
	if env := os.Getenv("CURRENCY_CONVERTER_PORT"); env != "" {
		return env
	}
	return "18880"
}
