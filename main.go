package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/berndhartzer/http-trace/report"
	"github.com/berndhartzer/http-trace/trace"
)

func main() {
	var method string
	var requestHeaders headerSlice
	var requestBody string
	var timeout int
	var suppressResponseHeaders, suppressResponseBody bool

	flag.StringVar(&method, "m", "GET", "The HTTP method to use")
	flag.Var(&requestHeaders, "H", "HTTP headers to send with the request")
	flag.StringVar(&requestBody, "d", "", "The HTTP request body data")
	flag.IntVar(&timeout, "t", 5, "Timeout for the HTTP request in seconds")
	flag.BoolVar(&suppressResponseHeaders, "suppress-headers", false, "Suppress the response headers in the output")
	flag.BoolVar(&suppressResponseBody, "suppress-body", false, "Suppress the response body in the output")

	flag.Parse()
	if flag.NArg() < 1 {
		exitWithError(fmt.Errorf("no url specified"))
	}
	url := flag.Arg(0)

	httpClient := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
	}

	req, err := http.NewRequest(method, url, strings.NewReader(requestBody))
	if err != nil {
		exitWithError(err)
	}

	tracedRequest := trace.New(httpClient, req)
	tracedRequest.SetHeaders(requestHeaders)
	err = tracedRequest.Execute()
	if err != nil {
		exitWithError(err)
	}

	resp := tracedRequest.GetResponse()
	responseBody := tracedRequest.GetResponseBody()
	timings := tracedRequest.GetTimings()

	presentation := &report.Presentation{
		SuppressHeaders: suppressResponseHeaders,
		SuppressBody:    suppressResponseBody,
	}

	output := report.New(req, resp, responseBody, timings, presentation)
	err = output.Build()
	if err != nil {
		exitWithError(err)
	}

	err = output.Print(os.Stdout)
	if err != nil {
		exitWithError(err)
	}
}

type headerSlice []string

func (h *headerSlice) String() string {
	return fmt.Sprintf("%s", *h)
}

func (h *headerSlice) Set(value string) error {
	*h = append(*h, value)
	return nil
}

func exitWithError(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err.Error())
	os.Exit(1)
}
