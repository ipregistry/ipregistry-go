[<img src="https://cdn.ipregistry.co/icons/favicon-96x96.png" alt="Ipregistry" width="64"/>](https://ipregistry.co/)
# Ipregistry Go Client Library

[![License](http://img.shields.io/:license-apache-blue.svg)](LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/ipregistry/ipregistry-go.svg)](https://pkg.go.dev/github.com/ipregistry/ipregistry-go)
[![Go CI](https://github.com/ipregistry/ipregistry-go/actions/workflows/go.yml/badge.svg)](https://github.com/ipregistry/ipregistry-go/actions/workflows/go.yml)
[![Lint](https://github.com/ipregistry/ipregistry-go/actions/workflows/lint.yml/badge.svg)](https://github.com/ipregistry/ipregistry-go/actions/workflows/lint.yml)

This is the official Go client library for the [Ipregistry](https://ipregistry.co) IP geolocation and threat data
API, allowing you to look up your own IP address or specified ones. Responses return multiple data points including
carrier, company, currency, location, time zone, threat information, and more. The library can also parse raw
User-Agent strings.

The library has **zero external dependencies** — it is built entirely on the Go standard library.

## Getting Started

You'll need an Ipregistry API key, which you can get along with 100,000 free lookups by signing up for a free account
at [https://ipregistry.co](https://ipregistry.co).

### Installation

```bash
go get github.com/ipregistry/ipregistry-go
```

Requires Go 1.23 or later.

```go
import ipregistry "github.com/ipregistry/ipregistry-go"
```

### Quick start

#### Single IP lookup

```go
package main

import (
	"context"
	"fmt"
	"log"

	ipregistry "github.com/ipregistry/ipregistry-go"
)

func main() {
	client := ipregistry.New("YOUR_API_KEY")
	defer client.Close()

	// Look up data for a given IPv4 or IPv6 address.
	// On the server side, retrieve the client IP from the request headers.
	info, err := client.Lookup(context.Background(), "54.85.132.205")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info.Location.Country.Name)
}
```

#### Origin IP lookup

To look up the IP address the request is sent from — no argument needed — use `LookupOrigin`. It returns a
`RequesterIPInfo`, which additionally carries parsed User-Agent data.

```go
origin, err := client.LookupOrigin(context.Background())
if err != nil {
	log.Fatal(err)
}
fmt.Println(origin.IP, origin.Location.Country.Name)
```

#### Batch IP lookup

`LookupBatch` resolves many IP addresses in a single request. Each entry may independently succeed or fail (for example
on an invalid address), so results are inspected element by element. Iterate with the `All` range-over-func, or index
with `At`:

```go
list, err := client.LookupBatch(context.Background(),
	[]string{"73.2.2.2", "8.8.8.8", "2001:67c:2e8:22::c100:68b"})
if err != nil {
	log.Fatal(err)
}

for info, err := range list.All() {
	if err != nil {
		// Handle a per-entry error (e.g. invalid IP address).
		log.Println("entry failed:", err)
		continue
	}
	fmt.Println(info.Location.Country.Name)
}
```

The Ipregistry API accepts up to 1024 IP addresses per request. `LookupBatch` transparently splits larger slices into
several requests, dispatched with bounded concurrency, and reassembles the results in input order — so you can pass an
arbitrarily long slice without hitting `TOO_MANY_IPS`. Tune the behavior when needed:

```go
client := ipregistry.New("YOUR_API_KEY",
	ipregistry.WithMaxBatchSize(1024),      // addresses per request (max/default: 1024)
	ipregistry.WithBatchConcurrency(4),     // concurrent sub-requests (default: 4; 1 = sequential)
)
```

Only cache misses are sent to the API; if a whole sub-request fails (network or API error), `LookupBatch` returns that
error, whereas an individual bad address surfaces as a per-entry error as shown above.

## Options

Lookups accept options that map to Ipregistry query parameters:

```go
info, err := client.Lookup(context.Background(), "8.8.8.8",
	ipregistry.WithHostname(true),                            // resolve reverse-DNS hostname
	ipregistry.WithFields("location.country.name,security"),  // select only these fields
)
```

| Option                    | Description                                                                          |
|---------------------------|--------------------------------------------------------------------------------------|
| `WithHostname(bool)`      | Enable reverse-DNS hostname resolution (disabled by default).                        |
| `WithFields(expression)`  | Restrict the response to the given [fields](https://ipregistry.co/docs/filtering-selecting-fields), reducing payload size. |
| `WithParam(name, value)`  | Set an arbitrary query parameter not covered by a dedicated helper.                  |

## Caching

Although the client has built-in support for in-memory caching, it is **disabled by default** to ensure data freshness.

To enable caching, pass an `InMemoryCache` when constructing the client:

```go
client := ipregistry.New("YOUR_API_KEY",
	ipregistry.WithCache(ipregistry.NewInMemoryCache()),
)
```

The in-memory cache is thread-safe and supports size- and time-based eviction (LRU with a TTL):

```go
cache := ipregistry.NewInMemoryCache(
	ipregistry.WithMaxSize(8192),          // maximum number of entries (default 4096)
	ipregistry.WithTTL(10*time.Minute),    // entry lifetime (default 10 minutes)
)

client := ipregistry.New("YOUR_API_KEY", ipregistry.WithCache(cache))
```

Origin (requester) lookups are never cached, because the requester IP is only known from the response. Batch lookups
transparently serve already-cached entries and only request the remainder from the API.

You can provide your own cache implementation by satisfying the `Cache` interface:

```go
type Cache interface {
	Get(key string) (*IPInfo, bool)
	Set(key string, value *IPInfo)
	Invalidate(key string)
	InvalidateAll()
}
```

## Retries

Failed requests are automatically retried with an exponential backoff. By default, up to 3 retries are performed on
transient network errors and 5xx server responses.

Because Ipregistry does not rate limit by default (rate limiting is opt-in per API key), retries on
_429 Too Many Requests_ responses are **disabled by default**. Enable them if your API key is configured with a rate
limit and you want the client to wait and retry (honoring the `Retry-After` header when present):

```go
client := ipregistry.New("YOUR_API_KEY",
	ipregistry.WithMaxRetries(3),                  // 0 disables retries entirely
	ipregistry.WithRetryInterval(time.Second),     // base backoff (interval * 2^attempt)
	ipregistry.WithRetryOnServerError(true),       // retry on 5xx (default: true)
	ipregistry.WithRetryOnTooManyRequests(true),   // retry on 429 (default: false)
)
```

## Context, timeouts, and concurrency

Every method takes a `context.Context`, so cancellation and deadlines compose naturally with the rest of your program.
There is no separate asynchronous API: a `Client` is safe for concurrent use, so run lookups in goroutines when you
need parallelism.

```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

info, err := client.Lookup(ctx, "8.8.8.8")
```

By default the client uses an `http.Client` with a 15-second timeout. Adjust it with `WithTimeout`, or supply your own
client for full control over connection pooling, proxying, TLS, or instrumentation:

```go
httpClient := &http.Client{ /* custom transport, proxy, TLS, timeout, ... */ }

client := ipregistry.New("YOUR_API_KEY", ipregistry.WithHTTPClient(httpClient))
```

When you supply your own client, you own its lifecycle: `Client.Close` does not touch it, and `WithTimeout` is ignored
in favor of your client's own settings.

## Errors

The library returns two typed error kinds, both matchable with `errors.As`:

- **`*APIError`** — the API reported a failure (e.g. insufficient credits, throttling, invalid input). It carries the
  raw `Code`, a typed `ErrorCode` (empty when the raw code is not recognized), a `Message`, and a `Resolution`.
- **`*ClientError`** — a client-side failure (network error, request cancellation, response decoding). The underlying
  cause is available via `errors.Unwrap` / `errors.Is`.

```go
info, err := client.Lookup(context.Background(), "8.8.8.8")

var apiErr *ipregistry.APIError
var clientErr *ipregistry.ClientError
switch {
case errors.As(err, &apiErr):
	if apiErr.ErrorCode == ipregistry.ErrorCodeInsufficientCredits {
		// handle exhausted credits
	} else if apiErr.ErrorCode == ipregistry.ErrorCodeTooManyRequests {
		// handle rate limiting
	}
case errors.As(err, &clientErr):
	// handle network / decoding error
}
```

The full list of error codes is documented at [ipregistry.co/docs/errors](https://ipregistry.co/docs/errors).

## Parsing User-Agents

Parse one or more raw User-Agent strings (such as the `User-Agent` header of an incoming request) into structured data:

```go
list, err := client.ParseUserAgents(context.Background(),
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) Chrome/120.0")
if err != nil {
	log.Fatal(err)
}
ua, err := list.At(0)
if err != nil {
	log.Fatal(err)
}
fmt.Println(ua.Name, ua.OperatingSystem.Name)
```

## Filtering bots

You might want to prevent Ipregistry API calls for crawlers or bots browsing your pages. To help identify bots from the
User-Agent, the library includes a lightweight helper:

```go
// For testing you can retrieve your current User-Agent from:
// https://api.ipregistry.co/user_agent?key=YOUR_API_KEY (look at the "user_agent" field)
if !ipregistry.IsBot(userAgentFromRequestHeader) {
	info, err := client.Lookup(context.Background(), clientIP)
	// ...
}
```

## Examples

Runnable examples live in the [`examples/`](examples) directory. Each is a standalone `main` package; set your key and
run it:

```bash
IPREGISTRY_API_KEY=YOUR_API_KEY go run ./examples/single
```

## Testing

The library ships with two tiers of tests:

- **Unit / behavior tests** run offline against an in-process `net/http/httptest` server — no API key or network is
  required. This is the default `go test ./...` and what CI runs (with the race detector and coverage).
- **System tests** exercise the live Ipregistry API. They live behind the `integration` build tag and are skipped
  unless `IPREGISTRY_API_KEY` is set (each successful lookup consumes credits):

  ```bash
  IPREGISTRY_API_KEY=YOUR_API_KEY go test -tags integration -run Integration ./...
  ```

Common tasks are wired through the [`Makefile`](Makefile): `make test`, `make race`, `make cover`, `make vet`,
`make fmtcheck`, `make lint`, and `make integration`.

## Other Libraries

There are official Ipregistry client libraries available for many languages including
[Java](https://github.com/ipregistry/ipregistry-java),
[Javascript](https://github.com/ipregistry/ipregistry-javascript),
[Python](https://github.com/ipregistry/ipregistry-python),
[Typescript](https://github.com/ipregistry/ipregistry-javascript) and more.

Are you looking for an official client with a programming language or framework we do not support yet?
[Let us know](mailto:support@ipregistry.co).

## License

This library is released under the [Apache 2.0 license](LICENSE).
