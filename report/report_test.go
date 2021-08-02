package report

import (
	"bytes"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/berndhartzer/http-trace/trace"
)

type testReport struct {
	presentation *Presentation
	expected     string
}

func TestReport(t *testing.T) {
	request, err := http.NewRequest(http.MethodGet, "https://thing.com", nil)
	if err != nil {
		t.Errorf("Error creating http request: %v", err)
	}
	request.Header.Set("X-Hello", "hi")

	expectedRequestHeadersOutput := `> GET thing.com HTTP/1.1
> X-Hello: hi
>
`
	response := &http.Response{
		Status: "200 OK",
		Header: map[string][]string{
			"Content-Type":  {"text/html; charset=utf-8"},
			"Vary":          {"Accept-Encoding"},
			"X-Some-Header": {"one", "two", "three"},
		},
	}

	expectedResponseStatusOutput := "< 200 OK\n"

	expectedResponseHeadersOutput := `< Content-Type: text/html; charset=utf-8
< Vary: Accept-Encoding
< X-Some-Header: onetwothree
`

	body := `<html>
<head>
<title>Welcome to nginx!</title>
</head>
<body bgcolor="white" text="black">
<center><h1>Welcome to the website!</h1></center>
</body>
</html>`

	timings := &trace.Timings{
		DNSDuration:             2293 * time.Microsecond,
		ConnectionDialDuration:  22664 * time.Microsecond,
		TLSDuration:             299741 * time.Microsecond,
		TotalConnectionDuration: 324931 * time.Microsecond,
		RequestWriteDuration:    48 * time.Microsecond,
		ResponseDelayDuration:   480966 * time.Microsecond,
		ResponseReadDuration:    22933 * time.Microsecond,
		TotalRequestDuration:    828987 * time.Microsecond,
	}

	expectedTraceOutput := `Trace
  Request
    Connection
      DNS Resolution:       2.29ms
      Connecting:          22.66ms
      TLS handshake:      299.74ms
    Connection total:     324.93ms

    Request write:          0.05ms
    Response delay:       480.97ms
    Response read:         22.93ms

  Request total:          828.99ms
`

	tests := map[string]testReport{
		"will output a full request and response with trace timings": {
			presentation: &Presentation{
				SuppressHeaders: false,
				SuppressBody:    false,
			},
			expected: fmt.Sprintf(
				"%s%s%s%s%s%s",
				expectedRequestHeadersOutput,
				expectedResponseStatusOutput,
				expectedResponseHeadersOutput,
				body,
				"\n\n",
				expectedTraceOutput,
			),
		},
		"will not output response headers if suppressed": {
			presentation: &Presentation{
				SuppressHeaders: true,
				SuppressBody:    false,
			},
			expected: fmt.Sprintf(
				"%s%s%s%s%s",
				expectedRequestHeadersOutput,
				expectedResponseStatusOutput,
				body,
				"\n\n",
				expectedTraceOutput,
			),
		},
		"will not output response body if suppressed": {
			presentation: &Presentation{
				SuppressHeaders: false,
				SuppressBody:    true,
			},
			expected: fmt.Sprintf(
				"%s%s%s%s%s",
				expectedRequestHeadersOutput,
				expectedResponseStatusOutput,
				expectedResponseHeadersOutput,
				"\n",
				expectedTraceOutput,
			),
		},
		"will not output response headers and body if suppressed": {
			presentation: &Presentation{
				SuppressHeaders: true,
				SuppressBody:    true,
			},
			expected: fmt.Sprintf(
				"%s%s%s%s",
				expectedRequestHeadersOutput,
				expectedResponseStatusOutput,
				"\n",
				expectedTraceOutput,
			),
		},
	}

	for name, cfg := range tests {
		cfg := cfg
		t.Run(name, func(t *testing.T) {
			report := New(request, response, body, timings, cfg.presentation)

			err = report.Build()
			if err != nil {
				t.Errorf("Error building report: %v", err)
			}

			output := &bytes.Buffer{}

			report.Print(output)

			if output.String() != cfg.expected {
				t.Errorf("report output incorrect: got\n%v\n want\n%v\n", output.String(), cfg.expected)
			}
		})
	}
}
