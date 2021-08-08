# http-trace

A HTTP tracing tool, created after re-visiting [this Stack Overflow question](https://stackoverflow.com/questions/18215389/how-do-i-measure-request-and-response-times-at-once-using-curl) one too many times. Send a HTTP request, get the response and a trace report back.

Heavily based off [rakyll/hey](https://github.com/rakyll/hey) which is fantastic and highly recommended if you need a tool to send load to a web server.

## Usage
```
Usage: http-trace [options...] <url>

Options:
-H
      HTTP headers to send with the request
-d
      The HTTP request body data
-m
      The HTTP method to use (default "GET")
-suppress-body
      Suppress the response body in the output
-suppress-headers
      Suppress the response headers in the output
-t
      Timeout for the HTTP request in seconds (default 5)
```

### Example request
Send a `GET` request to `https://pkg.go.dev/net/http/httptrace`, add a couple of headers, and suppress the response headers and body from the output:
```sh
http-trace -m GET -suppress-headers -suppress-body -H 'X-One: hello' -H 'X-Two: one two three' https://pkg.go.dev/net/http/httptrace
```

The output of that command would look like this:
```
> GET pkg.go.dev/net/http/httptrace HTTP/1.1
> X-One: hello
> X-Two: one two three
>
< 200 OK

Trace
  Request
    Connection
      DNS Resolution:     560.34ms
      Connecting:          24.93ms
      TLS handshake:      307.89ms
    Connection total:     893.50ms

    Request write:          0.05ms
    Response delay:       368.50ms
    Response read:         31.50ms

  Request total:         1293.66ms
```

## Trace metrics

```
Trace
  Request
    Connection
      DNS Resolution: DNS lookup duration
      Connecting:     Duration of time it takes to establish connection to destination server
      TLS handshake:  Duration of TLS handshake
    Connection total: Total connection setup (DNS lookup, Dial up and TLS) duration

    Request write:     Request write duration, from successful connection to completing write
    Response delay:    Delay duration between request being written and first byte of response being received
    Response read:     Response read duration, from receiving first byte of response to completing read

  Request total:       Total duration of the request (sending request, receiving and parsing response)
```

## Installation

You can download pre-built binaries for macOS or Linux from the [Releases Page](https://github.com/berndhartzer/http-trace/releases)

### Building from source

We can use the go tooling to build a binary:
```sh
# Build for your current environment
go build -o bin/http-trace

# macOS
GOOS=darwin GOARCH=amd64 go build -o ./bin/http-trace

# Linux
GOOS=linux GOARCH=amd64 go build -o ./bin/http-trace
```
