package trace

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptrace"
	"os"
	"strings"
	"time"
)

type Timings struct {
	getConnStart  time.Duration
	connectStart  time.Duration
	dnsStart      time.Duration
	tlsStart      time.Duration
	requestStart  time.Duration
	delayStart    time.Duration
	responseStart time.Duration

	DNSDuration             time.Duration // DNS lookup duration
	ConnectionDialDuration  time.Duration // Duration of time it takes to establish connection to destination server
	TLSDuration             time.Duration // Duration of TLS handshake
	TotalConnectionDuration time.Duration // Total connection setup (DNS lookup, Dial up and TLS) duration
	RequestWriteDuration    time.Duration // Request write duration, from successful connection to completing write
	ResponseDelayDuration   time.Duration // Delay duration between request being written and first byte of response being received
	ResponseReadDuration    time.Duration // Response read duration, from receiving first byte of response to completing read
	TotalRequestDuration    time.Duration // Total duration of the request (sending request, receiving and parsing response)
}

type Trace struct {
	timings      *Timings
	client       *http.Client
	request      *http.Request
	response     *http.Response
	responseBody string
}

func New(client *http.Client, request *http.Request) *Trace {
	timings := &Timings{}
	return &Trace{
		timings: timings,
		client:  client,
		request: request,
	}
}

func (t *Trace) SetHeaders(raw []string) {
	for _, full := range raw {
		split := strings.SplitN(full, ":", 2)
		headerTrim := strings.TrimSpace(split[0])
		valueTrim := strings.TrimSpace(split[1])
		t.request.Header.Set(headerTrim, valueTrim)
	}
}

func (t *Trace) Execute() error {
	var startTime = time.Now()
	timeSinceStart := func() time.Duration {
		return time.Since(startTime)
	}

	requestStartTime := timeSinceStart()

	trace := &httptrace.ClientTrace{
		GetConn: func(h string) {
			t.timings.getConnStart = timeSinceStart()
		},
		GotConn: func(connInfo httptrace.GotConnInfo) {
			if !connInfo.Reused {
				t.timings.TotalConnectionDuration = timeSinceStart() - t.timings.getConnStart
			}
			t.timings.requestStart = timeSinceStart()
		},
		GotFirstResponseByte: func() {
			t.timings.ResponseDelayDuration = timeSinceStart() - t.timings.delayStart
			t.timings.responseStart = timeSinceStart()
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			t.timings.dnsStart = timeSinceStart()
		},
		DNSDone: func(dnsInfo httptrace.DNSDoneInfo) {
			t.timings.DNSDuration = timeSinceStart() - t.timings.dnsStart
		},
		ConnectStart: func(network, addr string) {
			t.timings.connectStart = timeSinceStart()
		},
		ConnectDone: func(network, addr string, err error) {
			t.timings.ConnectionDialDuration = timeSinceStart() - t.timings.connectStart
		},
		TLSHandshakeStart: func() {
			t.timings.tlsStart = timeSinceStart()
		},
		TLSHandshakeDone: func(tlsConnState tls.ConnectionState, err error) {
			t.timings.TLSDuration = timeSinceStart() - t.timings.tlsStart
		},
		WroteRequest: func(w httptrace.WroteRequestInfo) {
			t.timings.RequestWriteDuration = timeSinceStart() - t.timings.requestStart
			t.timings.delayStart = timeSinceStart()
		},
	}

	t.request = t.request.WithContext(httptrace.WithClientTrace(t.request.Context(), trace))
	resp, err := t.client.Do(t.request)
	if err != nil {
		return fmt.Errorf("error sending request: %w", err)
	}

	responseBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		readingBodyError := fmt.Sprintf("Error reading response body: %v", err.Error())
		_, _ = fmt.Fprint(os.Stderr, readingBodyError+"\n")
		responseBodyBytes = []byte(readingBodyError)
	}
	responseBody := string(responseBodyBytes)
	resp.Body.Close()

	t.response = resp
	t.responseBody = responseBody

	finishTime := timeSinceStart()
	t.timings.ResponseReadDuration = finishTime - t.timings.responseStart
	t.timings.TotalRequestDuration = finishTime - requestStartTime

	return nil
}

func (t *Trace) GetResponse() *http.Response {
	return t.response
}

func (t *Trace) GetResponseBody() string {
	return t.responseBody
}

func (t *Trace) GetTimings() *Timings {
	return t.timings
}
