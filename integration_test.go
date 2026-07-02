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

//go:build integration

// These system tests run against the live Ipregistry API. They are excluded
// from the normal test run and only compile under the "integration" build tag:
//
//	IPREGISTRY_API_KEY=YOUR_API_KEY go test -tags integration -run Integration ./...
//
// A valid API key is required; each successful lookup consumes credits.
package ipregistry_test

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	ipregistry "github.com/ipregistry/ipregistry-go"
)

func newIntegrationClient(t *testing.T) *ipregistry.Client {
	t.Helper()
	key := os.Getenv("IPREGISTRY_API_KEY")
	if key == "" {
		t.Skip("set IPREGISTRY_API_KEY to run integration tests")
	}
	return ipregistry.New(key)
}

func integrationContext(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 30*time.Second)
}

func TestIntegrationLookup(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()
	ctx, cancel := integrationContext(t)
	defer cancel()

	info, err := client.Lookup(ctx, "8.8.8.8")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if info.IP != "8.8.8.8" {
		t.Errorf("ip = %q, want 8.8.8.8", info.IP)
	}
	if info.Location.Country.Code == "" {
		t.Error("expected a non-empty country code")
	}
	if info.Connection.ASN == nil {
		t.Error("expected a non-nil ASN for a well-known address")
	}
}

func TestIntegrationLookupOrigin(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()
	ctx, cancel := integrationContext(t)
	defer cancel()

	origin, err := client.LookupOrigin(ctx)
	if err != nil {
		t.Fatalf("LookupOrigin: %v", err)
	}
	if origin.IP == "" {
		t.Error("expected a non-empty origin IP")
	}
}

func TestIntegrationLookupBatch(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()
	ctx, cancel := integrationContext(t)
	defer cancel()

	ips := []string{"8.8.8.8", "1.1.1.1", "not-an-ip"}
	list, err := client.LookupBatch(ctx, ips)
	if err != nil {
		t.Fatalf("LookupBatch: %v", err)
	}
	if list.Len() != len(ips) {
		t.Fatalf("len = %d, want %d", list.Len(), len(ips))
	}

	if _, err := list.At(0); err != nil {
		t.Errorf("entry 0 should succeed: %v", err)
	}
	// The invalid address should surface as a per-entry APIError.
	if _, err := list.At(2); err == nil {
		t.Error("entry 2 (invalid IP) should have failed")
	} else {
		var aerr *ipregistry.APIError
		if !errors.As(err, &aerr) {
			t.Errorf("entry 2 err = %v, want *APIError", err)
		}
	}
}

func TestIntegrationInvalidIP(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()
	ctx, cancel := integrationContext(t)
	defer cancel()

	_, err := client.Lookup(ctx, "invalid")
	var aerr *ipregistry.APIError
	if !errors.As(err, &aerr) {
		t.Fatalf("err = %v, want *APIError", err)
	}
	if aerr.ErrorCode != ipregistry.ErrorCodeInvalidIPAddress {
		t.Errorf("error code = %q, want %q", aerr.ErrorCode, ipregistry.ErrorCodeInvalidIPAddress)
	}
}

func TestIntegrationParseUserAgents(t *testing.T) {
	client := newIntegrationClient(t)
	defer client.Close()
	ctx, cancel := integrationContext(t)
	defer cancel()

	list, err := client.ParseUserAgents(ctx,
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36")
	if err != nil {
		t.Fatalf("ParseUserAgents: %v", err)
	}
	ua, err := list.At(0)
	if err != nil {
		t.Fatalf("At(0): %v", err)
	}
	if ua.Name == "" {
		t.Error("expected a non-empty user-agent name")
	}
}
