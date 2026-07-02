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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// roundTripperFunc adapts a function to http.RoundTripper.
type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestBuildURL(t *testing.T) {
	c := New("KEY", WithBaseURL("https://example.test/")) // trailing slash must be trimmed
	if got := c.baseURL; got != "https://example.test" {
		t.Fatalf("baseURL = %q, want trailing slash trimmed", got)
	}

	if got := c.buildURL("8.8.8.8", url.Values{}); got != "https://example.test/8.8.8.8" {
		t.Errorf("single URL = %q", got)
	}
	if got := c.buildURL("", url.Values{}); got != "https://example.test/" {
		t.Errorf("origin URL = %q", got)
	}
	params := url.Values{}
	params.Set("hostname", "true")
	if got := c.buildURL("8.8.8.8", params); got != "https://example.test/8.8.8.8?hostname=true" {
		t.Errorf("URL with params = %q", got)
	}
}

func TestTransportErrorIsRetried(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"ip":"8.8.8.8"}`)
	}))
	defer srv.Close()

	var calls int
	transport := roundTripperFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls <= 2 {
			return nil, errors.New("simulated transport failure")
		}
		return http.DefaultTransport.RoundTrip(r)
	})

	client := New("KEY",
		WithBaseURL(srv.URL),
		WithHTTPClient(&http.Client{Transport: transport}),
		WithRetryInterval(time.Millisecond),
	)

	info, err := client.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if info.IP != "8.8.8.8" {
		t.Errorf("ip = %q", info.IP)
	}
	if calls != 3 {
		t.Errorf("transport calls = %d, want 3 (2 failures + 1 success)", calls)
	}
}

func TestCustomHTTPClientNotClosed(t *testing.T) {
	client := New("KEY", WithHTTPClient(&http.Client{}))
	if client.ownsHTTPClient {
		t.Error("client must not own a caller-provided HTTP client")
	}
	if err := client.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
	// Close is idempotent.
	if err := client.Close(); err != nil {
		t.Errorf("second Close: %v", err)
	}
}

func TestWithUserAgent(t *testing.T) {
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("User-Agent")
		io.WriteString(w, `{"ip":"8.8.8.8"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithUserAgent("my-app/1.0"))
	if _, err := client.Lookup(context.Background(), "8.8.8.8"); err != nil {
		t.Fatal(err)
	}
	if got != "my-app/1.0" {
		t.Errorf("User-Agent = %q, want my-app/1.0", got)
	}
}

func TestNonJSONErrorBodyFallsBackToGenericAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		io.WriteString(w, "502 Bad Gateway")
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := client.Lookup(context.Background(), "8.8.8.8")

	var aerr *APIError
	if !errors.As(err, &aerr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if aerr.Code != "" {
		t.Errorf("Code = %q, want empty for unparseable body", aerr.Code)
	}
	if aerr.Message == "" {
		t.Error("expected a non-empty fallback message")
	}
}

func TestParseUserAgentsNoArgsSendsEmptyArray(t *testing.T) {
	var body string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		body = string(b)
		io.WriteString(w, `{"results":[]}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	list, err := client.ParseUserAgents(context.Background())
	if err != nil {
		t.Fatalf("ParseUserAgents: %v", err)
	}
	if body != "[]" {
		t.Errorf("body = %q, want []", body)
	}
	if list.Len() != 0 {
		t.Errorf("len = %d, want 0", list.Len())
	}
}

func TestDecodeErrorIsClientError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `not json`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	_, err := client.Lookup(context.Background(), "8.8.8.8")

	var cerr *ClientError
	if !errors.As(err, &cerr) {
		t.Fatalf("err = %v, want *ClientError", err)
	}
}

func TestLookupAddr(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		io.WriteString(w, `{"ip":"8.8.8.8"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))

	info, err := client.LookupAddr(context.Background(), netip.MustParseAddr("8.8.8.8"))
	if err != nil {
		t.Fatalf("LookupAddr: %v", err)
	}
	if gotPath != "/8.8.8.8" {
		t.Errorf("path = %q, want /8.8.8.8", gotPath)
	}
	if info.IP != "8.8.8.8" {
		t.Errorf("ip = %q", info.IP)
	}

	// The zero Addr is rejected client-side without a request.
	if _, err := client.LookupAddr(context.Background(), netip.Addr{}); err == nil {
		t.Error("expected an error for the zero netip.Addr")
	} else {
		var cerr *ClientError
		if !errors.As(err, &cerr) {
			t.Errorf("err = %v, want *ClientError", err)
		}
	}
}

func TestLookupBatchAddr(t *testing.T) {
	var reqs int32
	srv := echoBatchServer(t, &reqs)
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	addrs := []netip.Addr{
		netip.MustParseAddr("1.1.1.1"),
		netip.MustParseAddr("2606:4700:4700::1111"),
	}

	list, err := client.LookupBatchAddr(context.Background(), addrs)
	if err != nil {
		t.Fatalf("LookupBatchAddr: %v", err)
	}
	if list.Len() != 2 {
		t.Fatalf("len = %d, want 2", list.Len())
	}
	for i, want := range []string{"1.1.1.1", "2606:4700:4700::1111"} {
		info, err := list.At(i)
		if err != nil {
			t.Fatalf("At(%d): %v", i, err)
		}
		if info.IP != want {
			t.Errorf("At(%d).IP = %q, want %q", i, info.IP, want)
		}
	}

	// An invalid address is rejected before any request is sent.
	if _, err := client.LookupBatchAddr(context.Background(), []netip.Addr{{}}); err == nil {
		t.Error("expected an error for an invalid netip.Addr")
	}
}

// echoBatchServer returns a server that echoes each posted IP back as a result,
// preserving order, and counts how many requests it received.
func echoBatchServer(t *testing.T, reqs *int32) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(reqs, 1)
		var ips []string
		if err := json.NewDecoder(r.Body).Decode(&ips); err != nil {
			t.Errorf("decode body: %v", err)
		}
		var sb strings.Builder
		sb.WriteString(`{"results":[`)
		for i, ip := range ips {
			if i > 0 {
				sb.WriteByte(',')
			}
			fmt.Fprintf(&sb, `{"ip":%q}`, ip)
		}
		sb.WriteString(`]}`)
		io.WriteString(w, sb.String())
	}))
}

func TestBatchChunking(t *testing.T) {
	var reqs int32
	srv := echoBatchServer(t, &reqs)
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithMaxBatchSize(2))
	ips := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3", "4.4.4.4", "5.5.5.5"}

	list, err := client.LookupBatch(context.Background(), ips)
	if err != nil {
		t.Fatalf("LookupBatch: %v", err)
	}
	// 5 IPs, chunk size 2 => 3 requests (2 + 2 + 1).
	if reqs != 3 {
		t.Errorf("requests = %d, want 3", reqs)
	}
	if list.Len() != len(ips) {
		t.Fatalf("len = %d, want %d", list.Len(), len(ips))
	}
	// Results must stay in input order despite concurrent dispatch.
	for i, want := range ips {
		info, err := list.At(i)
		if err != nil {
			t.Fatalf("At(%d): %v", i, err)
		}
		if info.IP != want {
			t.Errorf("At(%d).IP = %q, want %q", i, info.IP, want)
		}
	}
}

func TestBatchChunkingSequential(t *testing.T) {
	var reqs int32
	srv := echoBatchServer(t, &reqs)
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithMaxBatchSize(1), WithBatchConcurrency(1))
	ips := []string{"1.1.1.1", "2.2.2.2", "3.3.3.3"}

	list, err := client.LookupBatch(context.Background(), ips)
	if err != nil {
		t.Fatalf("LookupBatch: %v", err)
	}
	if reqs != 3 {
		t.Errorf("requests = %d, want 3", reqs)
	}
	for i, want := range ips {
		info, _ := list.At(i)
		if info.IP != want {
			t.Errorf("At(%d).IP = %q, want %q", i, info.IP, want)
		}
	}
}

func TestBatchChunkErrorPropagates(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var ips []string
		json.NewDecoder(r.Body).Decode(&ips)
		for _, ip := range ips {
			if ip == "boom" {
				w.WriteHeader(http.StatusInternalServerError)
				io.WriteString(w, `{"code":"INTERNAL","message":"boom"}`)
				return
			}
		}
		io.WriteString(w, `{"results":[{"ip":"`+ips[0]+`"}]}`)
	}))
	defer srv.Close()

	client := New("KEY",
		WithBaseURL(srv.URL),
		WithMaxBatchSize(1),
		WithMaxRetries(0), // fail fast; don't retry the 5xx chunk
	)

	_, err := client.LookupBatch(context.Background(), []string{"1.1.1.1", "boom", "2.2.2.2"})
	if err == nil {
		t.Fatal("expected the whole batch to fail when a chunk errors")
	}
	var aerr *APIError
	if !errors.As(err, &aerr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
}

func TestBatchEmptyInputSkipsAPI(t *testing.T) {
	var reqs int32
	srv := echoBatchServer(t, &reqs)
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	list, err := client.LookupBatch(context.Background(), nil)
	if err != nil {
		t.Fatalf("LookupBatch(nil): %v", err)
	}
	if list.Len() != 0 {
		t.Errorf("len = %d, want 0", list.Len())
	}
	if reqs != 0 {
		t.Errorf("requests = %d, want 0 (empty input must not hit the API)", reqs)
	}
}

func TestBatchAllCacheMissesRoundTrip(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"results":[{"ip":"8.8.8.8"},{"ip":"1.1.1.1"}]}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	list, err := client.LookupBatch(context.Background(), []string{"8.8.8.8", "1.1.1.1"})
	if err != nil {
		t.Fatalf("LookupBatch: %v", err)
	}
	if list.Len() != 2 {
		t.Fatalf("len = %d, want 2", list.Len())
	}
	for i, want := range []string{"8.8.8.8", "1.1.1.1"} {
		info, err := list.At(i)
		if err != nil {
			t.Fatalf("At(%d): %v", i, err)
		}
		if info.IP != want {
			t.Errorf("At(%d).IP = %q, want %q", i, info.IP, want)
		}
	}
}
