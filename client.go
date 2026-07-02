// Copyright 2019 Ipregistry (https://ipregistry.co).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ipregistry

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strconv"
	"sync"
	"time"
)

// Client sends requests to the Ipregistry API. Create one with New. A Client is
// safe for concurrent use by multiple goroutines.
type Client struct {
	apiKey  string
	baseURL string

	httpClient     *http.Client
	ownsHTTPClient bool
	timeout        time.Duration

	cache Cache

	maxRetries             int
	retryInterval          time.Duration
	retryOnServerError     bool
	retryOnTooManyRequests bool

	maxBatchSize     int
	batchConcurrency int

	userAgent string
}

// New creates a Client authenticating with the given API key. You can obtain a
// key, along with a generous free tier, at https://ipregistry.co.
//
// By default the client manages its own *http.Client with a 15-second timeout,
// retries transient failures up to three times, and performs no caching.
// Behavior is customized with Option values.
func New(apiKey string, opts ...Option) *Client {
	c := &Client{
		apiKey:                 apiKey,
		baseURL:                DefaultBaseURL,
		cache:                  noopCache{},
		timeout:                defaultTimeout,
		maxRetries:             defaultMaxRetries,
		retryInterval:          defaultRetryInterval,
		retryOnServerError:     defaultRetryOnServerError,
		retryOnTooManyRequests: defaultRetryOnTooManyRequests,
		maxBatchSize:           DefaultMaxBatchSize,
		batchConcurrency:       defaultBatchConcurrency,
		userAgent:              userAgent,
	}

	for _, opt := range opts {
		opt(c)
	}

	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: c.timeout}
		c.ownsHTTPClient = true
	}
	if c.cache == nil {
		c.cache = noopCache{}
	}

	return c
}

// Cache returns the cache used by the client.
func (c *Client) Cache() Cache {
	return c.cache
}

// Lookup returns the data associated with the given IP address. The ip argument
// must be a non-empty IPv4 or IPv6 address; to look up the requester's own IP,
// use LookupOrigin instead.
//
// When a cache is configured, a hit is returned without contacting the API.
func (c *Client) Lookup(ctx context.Context, ip string, opts ...LookupOption) (*IPInfo, error) {
	if ip == "" {
		return nil, &ClientError{Message: "ip must not be empty; use LookupOrigin for the requester IP"}
	}

	params := buildParams(opts)
	key := cacheKey(ip, params)
	if info, ok := c.cache.Get(key); ok {
		return info, nil
	}

	data, err := c.do(ctx, http.MethodGet, c.buildURL(ip, params), nil)
	if err != nil {
		return nil, err
	}

	info := new(IPInfo)
	if err := decode(data, info); err != nil {
		return nil, err
	}

	c.cache.Set(key, info)
	return info, nil
}

// LookupAddr returns the data associated with the given IP address. It is a
// typed convenience over Lookup for callers that already hold a netip.Addr
// (from net/netip); it fails fast with a *ClientError if addr is the zero value.
//
// Most callers receive IP addresses as strings — for example from a request's
// X-Forwarded-For header — and can use Lookup directly.
func (c *Client) LookupAddr(ctx context.Context, addr netip.Addr, opts ...LookupOption) (*IPInfo, error) {
	if !addr.IsValid() {
		return nil, &ClientError{Message: "invalid IP address"}
	}
	return c.Lookup(ctx, addr.String(), opts...)
}

// LookupOrigin returns the data associated with the IP address the request
// originates from, enriched with parsed User-Agent data. Origin lookups are
// never cached, because the requester IP is only known from the response.
func (c *Client) LookupOrigin(ctx context.Context, opts ...LookupOption) (*RequesterIPInfo, error) {
	params := buildParams(opts)

	data, err := c.do(ctx, http.MethodGet, c.buildURL("", params), nil)
	if err != nil {
		return nil, err
	}

	info := new(RequesterIPInfo)
	if err := decode(data, info); err != nil {
		return nil, err
	}
	return info, nil
}

// LookupBatch resolves several IP addresses in a single request. The returned
// IPInfoList preserves the order of ips, and each entry may independently
// succeed or fail. A non-nil error indicates the whole request failed (for
// example authentication or a network error), not the failure of an individual
// entry.
//
// Entries already present in the cache are served locally; only the remainder
// are requested from the API, and freshly resolved entries are cached.
func (c *Client) LookupBatch(ctx context.Context, ips []string, opts ...LookupOption) (*IPInfoList, error) {
	params := buildParams(opts)

	cached := make([]*IPInfo, len(ips))
	misses := make([]string, 0, len(ips))
	for i, ip := range ips {
		if info, ok := c.cache.Get(cacheKey(ip, params)); ok {
			cached[i] = info
		} else {
			misses = append(misses, ip)
		}
	}

	fresh, err := c.resolveMisses(ctx, misses, params)
	if err != nil {
		return nil, err
	}

	results := make([]IPInfoResult, len(ips))
	next := 0
	for i := range ips {
		if cached[i] != nil {
			results[i] = IPInfoResult{Info: cached[i]}
			continue
		}
		if next >= len(fresh.Results) {
			// Defensive: the API returned fewer results than requested.
			results[i] = IPInfoResult{Err: &APIError{Message: "missing result for requested IP address"}}
			continue
		}
		r := fresh.Results[next]
		next++
		results[i] = r
		if r.Info != nil {
			c.cache.Set(cacheKey(ips[i], params), r.Info)
		}
	}

	return &IPInfoList{Results: results}, nil
}

// LookupBatchAddr is the netip.Addr variant of LookupBatch. It fails fast with a
// *ClientError if any address is the zero value.
func (c *Client) LookupBatchAddr(ctx context.Context, addrs []netip.Addr, opts ...LookupOption) (*IPInfoList, error) {
	ips := make([]string, len(addrs))
	for i, addr := range addrs {
		if !addr.IsValid() {
			return nil, &ClientError{Message: "invalid IP address at index " + strconv.Itoa(i)}
		}
		ips[i] = addr.String()
	}
	return c.LookupBatch(ctx, ips, opts...)
}

// resolveMisses fetches fresh data for the cache-missed IP addresses. It sends a
// single request when the addresses fit within the API's per-request limit, and
// otherwise splits them into chunks dispatched with bounded concurrency. The
// returned results preserve the order of misses.
func (c *Client) resolveMisses(ctx context.Context, misses []string, params url.Values) (*IPInfoList, error) {
	if len(misses) == 0 {
		return &IPInfoList{}, nil
	}
	if len(misses) <= c.maxBatchSize {
		return c.doBatchRequest(ctx, misses, params)
	}
	return c.resolveChunks(ctx, misses, params)
}

// doBatchRequest performs a single POST batch request for the given addresses.
func (c *Client) doBatchRequest(ctx context.Context, ips []string, params url.Values) (*IPInfoList, error) {
	body, err := json.Marshal(ips)
	if err != nil {
		return nil, &ClientError{Message: "failed to encode request body", Err: err}
	}

	data, err := c.do(ctx, http.MethodPost, c.buildURL("", params), body)
	if err != nil {
		return nil, err
	}

	list := &IPInfoList{}
	if err := decode(data, list); err != nil {
		return nil, err
	}
	return list, nil
}

// resolveChunks splits misses into API-sized chunks, dispatches them with at
// most batchConcurrency in flight, and concatenates their results in order. If
// any chunk fails, the first error is returned and the remaining in-flight
// requests are cancelled.
func (c *Client) resolveChunks(ctx context.Context, misses []string, params url.Values) (*IPInfoList, error) {
	var chunks [][]string
	for start := 0; start < len(misses); start += c.maxBatchSize {
		end := min(start+c.maxBatchSize, len(misses))
		chunks = append(chunks, misses[start:end])
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	results := make([][]IPInfoResult, len(chunks))
	sem := make(chan struct{}, c.batchConcurrency)

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)

	for idx, chunk := range chunks {
		wg.Add(1)
		go func(idx int, chunk []string) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				return
			}

			list, err := c.doBatchRequest(ctx, chunk, params)
			if err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = err
					cancel() // stop the remaining chunks
				}
				mu.Unlock()
				return
			}
			results[idx] = list.Results
		}(idx, chunk)
	}
	wg.Wait()

	if firstErr != nil {
		return nil, firstErr
	}

	merged := make([]IPInfoResult, 0, len(misses))
	for _, chunkResults := range results {
		merged = append(merged, chunkResults...)
	}
	return &IPInfoList{Results: merged}, nil
}

// ParseUserAgents parses one or more raw User-Agent strings (such as the
// User-Agent header of an incoming HTTP request) into structured data. Results
// preserve the order of the input.
func (c *Client) ParseUserAgents(ctx context.Context, userAgents ...string) (*UserAgentList, error) {
	if userAgents == nil {
		userAgents = []string{}
	}
	body, err := json.Marshal(userAgents)
	if err != nil {
		return nil, &ClientError{Message: "failed to encode request body", Err: err}
	}

	data, err := c.do(ctx, http.MethodPost, c.baseURL+"/user_agent", body)
	if err != nil {
		return nil, err
	}

	list := new(UserAgentList)
	if err := decode(data, list); err != nil {
		return nil, err
	}
	return list, nil
}

// Close releases resources held by the client. When the client owns its
// *http.Client (the default), idle connections are closed. It is safe to call
// Close multiple times. A closed client should no longer be used.
func (c *Client) Close() error {
	if c.ownsHTTPClient {
		c.httpClient.CloseIdleConnections()
	}
	return nil
}

// buildURL builds the request URL for a single-IP or origin lookup. An empty ip
// targets the origin (requester) endpoint.
func (c *Client) buildURL(ip string, params url.Values) string {
	u := c.baseURL + "/" + ip
	if q := params.Encode(); q != "" {
		u += "?" + q
	}
	return u
}

// do performs an HTTP request with automatic retries and returns the raw 2xx
// response body. Non-2xx responses are converted to *APIError; transport and
// I/O failures to *ClientError.
func (c *Client) do(ctx context.Context, method, fullURL string, body []byte) ([]byte, error) {
	for attempt := 0; ; attempt++ {
		var reader io.Reader
		if body != nil {
			reader = bytes.NewReader(body)
		}

		req, err := http.NewRequestWithContext(ctx, method, fullURL, reader)
		if err != nil {
			return nil, &ClientError{Message: "failed to build request", Err: err}
		}
		req.Header.Set("Authorization", "ApiKey "+c.apiKey)
		req.Header.Set("User-Agent", c.userAgent)
		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/json")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if ctx.Err() != nil {
				return nil, &ClientError{Message: "request cancelled", Err: ctx.Err()}
			}
			// Transport errors are retried up to maxRetries regardless of the
			// retry-on-status flags, matching the reference implementation.
			if attempt < c.maxRetries {
				if berr := c.backoff(ctx, attempt, 0); berr != nil {
					return nil, berr
				}
				continue
			}
			return nil, &ClientError{Message: "request failed", Err: err}
		}

		data, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			if readErr != nil {
				if attempt < c.maxRetries {
					if berr := c.backoff(ctx, attempt, 0); berr != nil {
						return nil, berr
					}
					continue
				}
				return nil, &ClientError{Message: "failed to read response body", Err: readErr}
			}
			return data, nil
		}

		if c.shouldRetryStatus(resp.StatusCode) && attempt < c.maxRetries {
			retryAfter := parseRetryAfter(resp.Header.Get("Retry-After"))
			if berr := c.backoff(ctx, attempt, retryAfter); berr != nil {
				return nil, berr
			}
			continue
		}

		return nil, parseAPIError(data, resp.StatusCode)
	}
}

// shouldRetryStatus reports whether a non-2xx status is eligible for retry given
// the client's configuration.
func (c *Client) shouldRetryStatus(status int) bool {
	if status == http.StatusTooManyRequests {
		return c.retryOnTooManyRequests
	}
	if status >= 500 && status < 600 {
		return c.retryOnServerError
	}
	return false
}

// backoff waits before the next retry attempt, honoring an explicit Retry-After
// duration when positive and otherwise using exponential backoff. It returns a
// *ClientError if the context is cancelled while waiting.
func (c *Client) backoff(ctx context.Context, attempt int, retryAfter time.Duration) error {
	delay := retryAfter
	if delay <= 0 {
		shift := attempt
		if shift > 30 {
			shift = 30
		}
		delay = c.retryInterval * time.Duration(int64(1)<<uint(shift))
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return &ClientError{Message: "request cancelled during retry backoff", Err: ctx.Err()}
	case <-timer.C:
		return nil
	}
}

// decode unmarshals a successful response body into out.
func decode(data []byte, out any) error {
	if err := json.Unmarshal(data, out); err != nil {
		return &ClientError{Message: "failed to decode response", Err: err}
	}
	return nil
}

// parseAPIError converts a non-2xx response body into an *APIError, falling back
// to a generic message when the body is not a recognizable error payload.
func parseAPIError(data []byte, status int) error {
	var p apiErrorPayload
	if err := json.Unmarshal(data, &p); err != nil || p.Code == "" {
		return &APIError{Message: "unexpected HTTP status " + strconv.Itoa(status)}
	}
	return p.toAPIError()
}

// parseRetryAfter parses a Retry-After header expressed as an integer number of
// seconds. It returns 0 when the header is absent or not a valid non-negative
// integer (the HTTP-date form is not supported, matching the reference client).
func parseRetryAfter(value string) time.Duration {
	if value == "" {
		return 0
	}
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil || seconds < 0 {
		return 0
	}
	return time.Duration(seconds) * time.Second
}
