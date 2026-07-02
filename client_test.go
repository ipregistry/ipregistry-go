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
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestLookup(t *testing.T) {
	var gotAuth, gotUA, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotUA = r.Header.Get("User-Agent")
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, `{"ip":"8.8.8.8","type":"IPv4","location":{"country":{"name":"United States","code":"US"}},"connection":{"asn":15169,"organization":"Google"}}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	defer client.Close()

	info, err := client.Lookup(context.Background(), "8.8.8.8", WithHostname(true), WithFields("location"))
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}

	if gotAuth != "ApiKey KEY" {
		t.Errorf("Authorization = %q, want %q", gotAuth, "ApiKey KEY")
	}
	if !strings.HasPrefix(gotUA, "IpregistryClient/Go/") {
		t.Errorf("User-Agent = %q, want IpregistryClient/Go/ prefix", gotUA)
	}
	if gotPath != "/8.8.8.8" {
		t.Errorf("path = %q, want /8.8.8.8", gotPath)
	}
	if !strings.Contains(gotQuery, "hostname=true") || !strings.Contains(gotQuery, "fields=location") {
		t.Errorf("query = %q, missing options", gotQuery)
	}
	if info.Location.Country.Name != "United States" {
		t.Errorf("country = %q, want United States", info.Location.Country.Name)
	}
	if info.Connection.ASN == nil || *info.Connection.ASN != 15169 {
		t.Errorf("asn = %v, want 15169", info.Connection.ASN)
	}
	if info.Type != IPTypeIPv4 {
		t.Errorf("type = %q, want IPv4", info.Type)
	}
}

func TestLookupEmptyIP(t *testing.T) {
	client := New("KEY")
	_, err := client.Lookup(context.Background(), "")
	var cerr *ClientError
	if !errors.As(err, &cerr) {
		t.Fatalf("err = %v, want *ClientError", err)
	}
}

func TestLookupOrigin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			t.Errorf("path = %q, want /", r.URL.Path)
		}
		io.WriteString(w, `{"ip":"1.2.3.4","user_agent":{"name":"Chrome","os":{"name":"Linux"}}}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	origin, err := client.LookupOrigin(context.Background())
	if err != nil {
		t.Fatalf("LookupOrigin: %v", err)
	}
	if origin.IP != "1.2.3.4" {
		t.Errorf("ip = %q, want 1.2.3.4", origin.IP)
	}
	if origin.UserAgent == nil || origin.UserAgent.Name != "Chrome" {
		t.Fatalf("user agent = %v, want Chrome", origin.UserAgent)
	}
	if origin.UserAgent.OperatingSystem.Name != "Linux" {
		t.Errorf("os = %q, want Linux", origin.UserAgent.OperatingSystem.Name)
	}
}

func TestLookupBatch(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want POST", r.Method)
		}
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{"results":[{"ip":"8.8.8.8","location":{"country":{"name":"United States"}}},{"code":"INVALID_IP_ADDRESS","message":"Invalid IP","resolution":"Fix it"}]}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	list, err := client.LookupBatch(context.Background(), []string{"8.8.8.8", "bogus"})
	if err != nil {
		t.Fatalf("LookupBatch: %v", err)
	}
	if gotBody != `["8.8.8.8","bogus"]` {
		t.Errorf("body = %q, want JSON array of IPs", gotBody)
	}
	if list.Len() != 2 {
		t.Fatalf("len = %d, want 2", list.Len())
	}

	info, err := list.At(0)
	if err != nil {
		t.Fatalf("At(0): %v", err)
	}
	if info.Location.Country.Name != "United States" {
		t.Errorf("country = %q", info.Location.Country.Name)
	}

	_, err = list.At(1)
	var aerr *APIError
	if !errors.As(err, &aerr) {
		t.Fatalf("At(1) err = %v, want *APIError", err)
	}
	if aerr.ErrorCode != ErrorCodeInvalidIPAddress {
		t.Errorf("error code = %q, want %q", aerr.ErrorCode, ErrorCodeInvalidIPAddress)
	}

	// Iterator should visit both entries.
	var ok, failed int
	for info, err := range list.All() {
		if err != nil {
			failed++
		} else if info != nil {
			ok++
		}
	}
	if ok != 1 || failed != 1 {
		t.Errorf("iterator ok=%d failed=%d, want 1/1", ok, failed)
	}
}

func TestParseUserAgents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/user_agent" {
			t.Errorf("path = %q, want /user_agent", r.URL.Path)
		}
		io.WriteString(w, `{"results":[{"name":"Chrome","type":"browser"}]}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	list, err := client.ParseUserAgents(context.Background(), "Mozilla/5.0")
	if err != nil {
		t.Fatalf("ParseUserAgents: %v", err)
	}
	ua, err := list.At(0)
	if err != nil {
		t.Fatalf("At(0): %v", err)
	}
	if ua.Name != "Chrome" {
		t.Errorf("name = %q, want Chrome", ua.Name)
	}
}

func TestAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"code":"INSUFFICIENT_CREDITS","message":"Out of credits","resolution":"Top up"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	_, err := client.Lookup(context.Background(), "8.8.8.8")

	var aerr *APIError
	if !errors.As(err, &aerr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if aerr.ErrorCode != ErrorCodeInsufficientCredits {
		t.Errorf("error code = %q", aerr.ErrorCode)
	}
	if !strings.Contains(aerr.Error(), "Out of credits") {
		t.Errorf("Error() = %q", aerr.Error())
	}
}

func TestRetryOnServerError(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		io.WriteString(w, `{"ip":"8.8.8.8"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithRetryInterval(time.Millisecond))
	info, err := client.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if info.IP != "8.8.8.8" {
		t.Errorf("ip = %q", info.IP)
	}
	if calls != 3 {
		t.Errorf("calls = %d, want 3", calls)
	}
}

func TestNoRetryWhenDisabled(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
		io.WriteString(w, `{"code":"INTERNAL","message":"boom"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithMaxRetries(0))
	_, err := client.Lookup(context.Background(), "8.8.8.8")
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1", calls)
	}
}

func TestRetryOnTooManyRequestsDisabledByDefault(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusTooManyRequests)
		io.WriteString(w, `{"code":"TOO_MANY_REQUESTS","message":"slow down"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithRetryInterval(time.Millisecond))
	_, err := client.Lookup(context.Background(), "8.8.8.8")
	if err == nil {
		t.Fatal("expected error")
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (429 retry disabled by default)", calls)
	}
}

func TestContextCancellation(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		io.WriteString(w, `{"ip":"8.8.8.8"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL))
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()

	_, err := client.Lookup(ctx, "8.8.8.8")
	var cerr *ClientError
	if !errors.As(err, &cerr) {
		t.Fatalf("err = %v, want *ClientError", err)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("err should wrap context.DeadlineExceeded, got %v", err)
	}
}

func TestCacheServesRepeatLookups(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		io.WriteString(w, `{"ip":"8.8.8.8"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithCache(NewInMemoryCache()))
	for i := 0; i < 3; i++ {
		if _, err := client.Lookup(context.Background(), "8.8.8.8"); err != nil {
			t.Fatalf("Lookup: %v", err)
		}
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (subsequent lookups cached)", calls)
	}
}

func TestBatchUsesCacheAndSkipsAPIWhenFullyCached(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		io.WriteString(w, `{"results":[{"ip":"8.8.8.8"},{"ip":"1.1.1.1"}]}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithCache(NewInMemoryCache()))
	ips := []string{"8.8.8.8", "1.1.1.1"}
	if _, err := client.LookupBatch(context.Background(), ips); err != nil {
		t.Fatalf("first batch: %v", err)
	}
	// Second call is fully cached and must not hit the API.
	list, err := client.LookupBatch(context.Background(), ips)
	if err != nil {
		t.Fatalf("second batch: %v", err)
	}
	if calls != 1 {
		t.Errorf("calls = %d, want 1 (second batch fully cached)", calls)
	}
	if list.Len() != 2 {
		t.Errorf("len = %d, want 2", list.Len())
	}
}

func TestOriginLookupNotCached(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		io.WriteString(w, `{"ip":"1.2.3.4"}`)
	}))
	defer srv.Close()

	client := New("KEY", WithBaseURL(srv.URL), WithCache(NewInMemoryCache()))
	client.LookupOrigin(context.Background())
	client.LookupOrigin(context.Background())
	if calls != 2 {
		t.Errorf("calls = %d, want 2 (origin lookups are never cached)", calls)
	}
}
