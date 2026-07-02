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
	"testing"
	"time"
)

func TestIsBot(t *testing.T) {
	cases := map[string]bool{
		"Googlebot/2.1 (+http://www.google.com/bot.html)": true,
		"Mozilla/5.0 (compatible; bingbot/2.0)":           true,
		"Baiduspider":                                     true,
		"Yahoo! Slurp":                                    true,
		"Mozilla/5.0 (Windows NT 10.0) Chrome/120.0":      false,
		"": false,
	}
	for ua, want := range cases {
		if got := IsBot(ua); got != want {
			t.Errorf("IsBot(%q) = %v, want %v", ua, got, want)
		}
	}
}

func TestParseErrorCode(t *testing.T) {
	if got := ParseErrorCode("insufficient_credits"); got != ErrorCodeInsufficientCredits {
		t.Errorf("got %q, want %q", got, ErrorCodeInsufficientCredits)
	}
	if got := ParseErrorCode("  TOO_MANY_IPS  "); got != ErrorCodeTooManyIPs {
		t.Errorf("got %q, want %q", got, ErrorCodeTooManyIPs)
	}
	if got := ParseErrorCode("SOMETHING_NEW"); got != "" {
		t.Errorf("unknown code should yield empty, got %q", got)
	}
	if got := ParseErrorCode(""); got != "" {
		t.Errorf("empty code should yield empty, got %q", got)
	}
}

func TestCacheKeyDeterministic(t *testing.T) {
	a := buildParams([]LookupOption{WithFields("location"), WithHostname(true)})
	b := buildParams([]LookupOption{WithHostname(true), WithFields("location")})
	if cacheKey("8.8.8.8", a) != cacheKey("8.8.8.8", b) {
		t.Error("cache key should not depend on option ordering")
	}
	if cacheKey("8.8.8.8", nil) != "8.8.8.8" {
		t.Error("cache key without params should be the bare IP")
	}
}

func TestInMemoryCacheEviction(t *testing.T) {
	c := NewInMemoryCache(WithMaxSize(2))
	c.Set("a", &IPInfo{IP: "a"})
	c.Set("b", &IPInfo{IP: "b"})
	// Access "a" so "b" becomes least recently used.
	c.Get("a")
	c.Set("c", &IPInfo{IP: "c"})

	if _, ok := c.Get("b"); ok {
		t.Error("b should have been evicted")
	}
	if _, ok := c.Get("a"); !ok {
		t.Error("a should still be present")
	}
	if _, ok := c.Get("c"); !ok {
		t.Error("c should be present")
	}
}

func TestInMemoryCacheExpiry(t *testing.T) {
	c := NewInMemoryCache(WithTTL(time.Minute))
	now := time.Unix(0, 0)
	c.now = func() time.Time { return now }

	c.Set("k", &IPInfo{IP: "k"})
	if _, ok := c.Get("k"); !ok {
		t.Fatal("entry should be present before expiry")
	}

	now = now.Add(2 * time.Minute)
	if _, ok := c.Get("k"); ok {
		t.Error("entry should be expired")
	}
}

func TestInMemoryCacheInvalidate(t *testing.T) {
	c := NewInMemoryCache()
	c.Set("a", &IPInfo{IP: "a"})
	c.Set("b", &IPInfo{IP: "b"})

	c.Invalidate("a")
	if _, ok := c.Get("a"); ok {
		t.Error("a should be gone")
	}

	c.InvalidateAll()
	if c.Len() != 0 {
		t.Errorf("len = %d, want 0 after InvalidateAll", c.Len())
	}
}

func TestParseRetryAfter(t *testing.T) {
	if d := parseRetryAfter("5"); d != 5*time.Second {
		t.Errorf("got %v, want 5s", d)
	}
	if d := parseRetryAfter(""); d != 0 {
		t.Errorf("empty should be 0, got %v", d)
	}
	if d := parseRetryAfter("Wed, 21 Oct 2015 07:28:00 GMT"); d != 0 {
		t.Errorf("HTTP-date form unsupported, want 0, got %v", d)
	}
	if d := parseRetryAfter("-3"); d != 0 {
		t.Errorf("negative should be 0, got %v", d)
	}
}
