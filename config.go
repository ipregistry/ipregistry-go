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
	"net/http"
	"time"
)

// DefaultBaseURL is the base URL of the Ipregistry API used unless overridden
// with WithBaseURL.
const DefaultBaseURL = "https://api.ipregistry.co"

// DefaultMaxBatchSize is the maximum number of IP addresses Ipregistry accepts
// in a single batch request. LookupBatch transparently splits larger slices
// into several requests so callers never have to.
const DefaultMaxBatchSize = 1024

// Default client settings.
const (
	defaultTimeout                = 15 * time.Second
	defaultMaxRetries             = 3
	defaultRetryInterval          = time.Second
	defaultRetryOnServerError     = true
	defaultRetryOnTooManyRequests = false
	defaultBatchConcurrency       = 4
)

// Option configures a Client. Pass options to New.
type Option func(*Client)

// WithBaseURL overrides the API base URL. This is mainly useful for testing or
// pointing at a private deployment. A trailing slash is trimmed.
func WithBaseURL(baseURL string) Option {
	return func(c *Client) {
		for len(baseURL) > 0 && baseURL[len(baseURL)-1] == '/' {
			baseURL = baseURL[:len(baseURL)-1]
		}
		c.baseURL = baseURL
	}
}

// WithHTTPClient supplies a custom *http.Client, giving full control over
// connection pooling, proxying, TLS, and instrumentation. When set, the
// client's own Timeout takes precedence over WithTimeout, and the caller
// retains ownership: Client.Close does not close idle connections on a
// caller-provided client.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		if httpClient != nil {
			c.httpClient = httpClient
			c.ownsHTTPClient = false
		}
	}
}

// WithCache enables response caching using the supplied Cache. By default no
// cache is used so that data is never stale. Passing nil is a no-op.
func WithCache(cache Cache) Option {
	return func(c *Client) {
		if cache != nil {
			c.cache = cache
		}
	}
}

// WithTimeout sets the per-request timeout applied to the default HTTP client.
// It is ignored when a custom client is provided with WithHTTPClient. A value
// <= 0 disables the client-level timeout (rely on the request context instead).
func WithTimeout(d time.Duration) Option {
	return func(c *Client) {
		c.timeout = d
	}
}

// WithMaxRetries sets the maximum number of automatic retries performed in
// addition to the initial attempt. Set to 0 to disable retries. Defaults to 3.
func WithMaxRetries(n int) Option {
	return func(c *Client) {
		if n >= 0 {
			c.maxRetries = n
		}
	}
}

// WithRetryInterval sets the base backoff between retries. Successive retries
// use an exponentially increasing delay (interval * 2^attempt). When a 429
// response carries a Retry-After header, that value takes precedence. Defaults
// to 1 second.
func WithRetryInterval(d time.Duration) Option {
	return func(c *Client) {
		if d > 0 {
			c.retryInterval = d
		}
	}
}

// WithRetryOnServerError controls whether 5xx responses (and transient network
// errors) are retried. Defaults to true.
func WithRetryOnServerError(enabled bool) Option {
	return func(c *Client) {
		c.retryOnServerError = enabled
	}
}

// WithRetryOnTooManyRequests controls whether 429 Too Many Requests responses
// are retried, honoring the Retry-After header when present. Ipregistry does
// not rate limit by default (it is opt-in per API key), so this defaults to
// false.
func WithRetryOnTooManyRequests(enabled bool) Option {
	return func(c *Client) {
		c.retryOnTooManyRequests = enabled
	}
}

// WithUserAgent overrides the User-Agent header sent with requests.
func WithUserAgent(ua string) Option {
	return func(c *Client) {
		if ua != "" {
			c.userAgent = ua
		}
	}
}

// WithMaxBatchSize sets the maximum number of IP addresses sent in a single
// batch request. LookupBatch splits larger slices into this many addresses per
// request. Values are capped at DefaultMaxBatchSize (the API limit); a value
// <= 0 leaves the default.
func WithMaxBatchSize(n int) Option {
	return func(c *Client) {
		if n > 0 && n <= DefaultMaxBatchSize {
			c.maxBatchSize = n
		}
	}
}

// WithBatchConcurrency sets how many batch sub-requests LookupBatch dispatches
// concurrently when a slice is large enough to be split into chunks. A value
// <= 0 leaves the default (4). Set it to 1 for strictly sequential dispatch,
// which is gentler on a rate-limited API key.
func WithBatchConcurrency(n int) Option {
	return func(c *Client) {
		if n > 0 {
			c.batchConcurrency = n
		}
	}
}
