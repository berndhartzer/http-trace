package trace

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type testTrace struct {
	requestMethod          string
	requestHeadersPrepared map[string]string
	requestBody            string
	serverDelayMS          int
	responseStatusCode     int
	responseBody           []byte
}

func TestTrace(t *testing.T) {
	tests := map[string]testTrace{
		"will trace a request": {
			requestMethod:          http.MethodGet,
			requestHeadersPrepared: nil,
			requestBody:            "",
			serverDelayMS:          100,
			responseStatusCode:     http.StatusOK,
			responseBody:           nil,
		},
		"can send a request with a body": {
			requestMethod:          http.MethodPost,
			requestHeadersPrepared: nil,
			requestBody:            `{"something": "hello", "trueThing": true, "n": 5}`,
			serverDelayMS:          200,
			responseStatusCode:     http.StatusOK,
			responseBody:           nil,
		},
		"can send a request with additional headers": {
			requestMethod: http.MethodGet,
			requestHeadersPrepared: map[string]string{
				"Content-Type":     "something",
				"X-Something-Else": "hello",
			},
			requestBody:        "",
			serverDelayMS:      300,
			responseStatusCode: http.StatusCreated,
			responseBody:       []byte("ok"),
		},
		"will read and return the response body": {
			requestMethod:          http.MethodPut,
			requestHeadersPrepared: nil,
			requestBody:            "things=hello&another=55",
			serverDelayMS:          200,
			responseStatusCode:     http.StatusOK,
			responseBody:           []byte(`{"got_it": "yep"}`),
		},
		"can handle non-200 responses": {
			requestMethod:          http.MethodGet,
			requestHeadersPrepared: nil,
			requestBody:            "",
			serverDelayMS:          100,
			responseStatusCode:     http.StatusNotFound,
			responseBody:           nil,
		},
	}

	for name, cfg := range tests {
		cfg := cfg
		t.Run(name, func(t *testing.T) {
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Duration(cfg.serverDelayMS) * time.Millisecond)

				if cfg.requestHeadersPrepared != nil {
					for h, v := range cfg.requestHeadersPrepared {
						receivedHeader, ok := r.Header[h]
						if !ok {
							t.Errorf("Missing expected http header: %s", h)
						}
						if v != receivedHeader[0] {
							t.Errorf("request header %s value incorrect: got %v, want %v", h, receivedHeader[0], v)
						}
					}
				}

				if cfg.requestBody != "" {
					body, err := io.ReadAll(r.Body)
					if err != nil {
						t.Errorf("Error reading request body: %v", err)
					}

					if !bytes.Equal(body, []byte(cfg.requestBody)) {
						t.Errorf("Incorrect http request body: got %v, want %v", string(body), cfg.requestBody)
					}
				}

				w.WriteHeader(cfg.responseStatusCode)
				w.Write(cfg.responseBody)
			}))
			defer server.Close()

			// https://stackoverflow.com/questions/36340396/how-can-i-test-https-endpoints-in-go
			// get the test servers certificates
			certs := x509.NewCertPool()
			for _, c := range server.TLS.Certificates {
				roots, err := x509.ParseCertificates(c.Certificate[len(c.Certificate)-1])
				if err != nil {
					// log.Fatalf("error parsing server's root cert: %v", err)
				}
				for _, root := range roots {
					certs.AddCert(root)
				}
			}

			// configure the http client to use the test servers certificates
			httpClient := &http.Client{
				Timeout: time.Duration(1) * time.Second,
				Transport: &http.Transport{
					TLSClientConfig: &tls.Config{
						RootCAs: certs,
					},
				},
			}

			var requestBody io.Reader
			if cfg.requestBody != "" {
				requestBody = strings.NewReader(cfg.requestBody)
			}

			request, err := http.NewRequest(cfg.requestMethod, server.URL, requestBody)
			if err != nil {
				t.Errorf("Error creating http request: %v", err)
			}

			rawRequestHeaders := []string{}
			for h, v := range cfg.requestHeadersPrepared {
				rawRequestHeaders = append(rawRequestHeaders, fmt.Sprintf("%s: %s", h, v))
			}

			tracedRequest := New(httpClient, request)
			tracedRequest.SetHeaders(rawRequestHeaders)
			err = tracedRequest.Execute()
			if err != nil {
				t.Errorf("Error doing traced request: %v", err)
			}

			resp := tracedRequest.GetResponse()
			responseBody := tracedRequest.GetResponseBody()
			timings := tracedRequest.GetTimings()

			if resp.StatusCode != cfg.responseStatusCode {
				t.Errorf("Unexpected http response status code: got %v, want %v", resp.StatusCode, cfg.responseStatusCode)
			}

			expectedResponseBody := string(cfg.responseBody)
			if responseBody != expectedResponseBody {
				t.Errorf("Unexpected http response body: got %v, want %v", responseBody, expectedResponseBody)
			}

			// DNS lookup will be 0 due to server being local
			if timings.DNSDuration != time.Duration(0) {
				t.Errorf("Unexpected DNSDuration value: got %v, want 0s", timings.DNSDuration)
			}

			// For the rest of the timings we will check that they fall within some
			// reasonable range
			err = timeDurationIsInRange(t, timings.ConnectionDialDuration, time.Microsecond, 300, 200)
			if err != nil {
				t.Errorf("ConnectionDialDuration not in range: %v", err)
			}

			err = timeDurationIsInRange(t, timings.TLSDuration, time.Millisecond, 2, 2)
			if err != nil {
				t.Errorf("TLSDuration not in range: %v", err)
			}

			err = timeDurationIsInRange(t, timings.TotalConnectionDuration, time.Millisecond, 3, 2)
			if err != nil {
				t.Errorf("TotalConnectionDuration not in range: %v", err)
			}

			err = timeDurationIsInRange(t, timings.RequestWriteDuration, time.Microsecond, 150, 150)
			if err != nil {
				t.Errorf("RequestWriteDuration not in range: %v", err)
			}

			err = timeDurationIsInRange(t, timings.ResponseDelayDuration, time.Millisecond, cfg.serverDelayMS, 20)
			if err != nil {
				t.Errorf("ResponseDelayDuration not in range: %v", err)
			}

			err = timeDurationIsInRange(t, timings.ResponseReadDuration, time.Microsecond, 150, 150)
			if err != nil {
				t.Errorf("ResponseReadDuration not in range: %v", err)
			}

			err = timeDurationIsInRange(t, timings.TotalRequestDuration, time.Millisecond, cfg.serverDelayMS, 20)
			if err != nil {
				t.Errorf("TotalRequestDuration not in range: %v", err)
			}
		})
	}
}

func timeDurationIsInRange(t *testing.T, d, resolution time.Duration, expected, margin int) error {
	t.Helper()

	if d == time.Duration(0) {
		return fmt.Errorf("duration is 0")
	}

	lower := (time.Duration(expected) * resolution) - (time.Duration(margin) * resolution)
	upper := (time.Duration(expected) * resolution) + (time.Duration(margin) * resolution)
	if d < lower {
		return fmt.Errorf("duration too low: got %v, want at least: %v", d, lower)
	}
	if d > upper {
		return fmt.Errorf("duration too high: got %v, want at most: %v", d, upper)
	}

	return nil
}

type testTraceTimeout struct {
	timeout              time.Duration
	serverDelay          time.Duration
	expectedRequestError bool
}

func TestTraceTimeout(t *testing.T) {
	tests := map[string]testTraceTimeout{
		"will timeout if response exceeds timeout": {
			timeout:              200 * time.Millisecond,
			serverDelay:          500 * time.Millisecond,
			expectedRequestError: true,
		},
		"will not timeout if response within limits of timeout": {
			timeout:              500 * time.Millisecond,
			serverDelay:          100 * time.Millisecond,
			expectedRequestError: false,
		},
	}

	for name, cfg := range tests {
		cfg := cfg
		t.Run(name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(cfg.serverDelay)
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			httpClient := &http.Client{
				Timeout: cfg.timeout,
			}

			request, err := http.NewRequest(http.MethodGet, server.URL, nil)
			if err != nil {
				t.Errorf("Error creating http request: %v", err)
			}

			tracedRequest := New(httpClient, request)
			err = tracedRequest.Execute()

			if cfg.expectedRequestError {
				if err == nil {
					t.Error("Expected http request error")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected http request error: %v", err)
				}
			}
		})
	}
}
