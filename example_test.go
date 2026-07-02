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

package ipregistry_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	ipregistry "github.com/ipregistry/ipregistry-go"
)

func ExampleClient_Lookup() {
	client := ipregistry.New("YOUR_API_KEY")
	defer client.Close()

	info, err := client.Lookup(context.Background(), "8.8.8.8")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info.Location.Country.Name)
}

func ExampleClient_LookupOrigin() {
	client := ipregistry.New("YOUR_API_KEY")
	defer client.Close()

	origin, err := client.LookupOrigin(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(origin.IP, origin.Location.Country.Name)
}

func ExampleClient_LookupBatch() {
	client := ipregistry.New("YOUR_API_KEY")
	defer client.Close()

	list, err := client.LookupBatch(context.Background(),
		[]string{"73.2.2.2", "8.8.8.8", "2001:67c:2e8:22::c100:68b"})
	if err != nil {
		log.Fatal(err)
	}

	for info, err := range list.All() {
		if err != nil {
			log.Println("entry failed:", err)
			continue
		}
		fmt.Println(info.Location.Country.Name)
	}
}

func ExampleClient_Lookup_withOptions() {
	client := ipregistry.New("YOUR_API_KEY")
	defer client.Close()

	info, err := client.Lookup(context.Background(), "8.8.8.8",
		ipregistry.WithHostname(true),
		ipregistry.WithFields("location.country.name,security"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(info.Hostname, info.Security.IsThreat)
}

func ExampleClient_ParseUserAgents() {
	client := ipregistry.New("YOUR_API_KEY")
	defer client.Close()

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
}

func ExampleIsBot() {
	userAgent := "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)"
	if ipregistry.IsBot(userAgent) {
		fmt.Println("skip lookup for bot traffic")
	}
	// Output: skip lookup for bot traffic
}

func ExampleClient_errorHandling() {
	client := ipregistry.New("YOUR_API_KEY")
	defer client.Close()

	_, err := client.Lookup(context.Background(), "8.8.8.8")

	var apiErr *ipregistry.APIError
	var clientErr *ipregistry.ClientError
	switch {
	case errors.As(err, &apiErr):
		if apiErr.ErrorCode == ipregistry.ErrorCodeInsufficientCredits {
			fmt.Println("out of credits")
		}
	case errors.As(err, &clientErr):
		fmt.Println("network or decoding error:", clientErr)
	}
}

func ExampleNew_caching() {
	client := ipregistry.New("YOUR_API_KEY",
		ipregistry.WithCache(ipregistry.NewInMemoryCache(
			ipregistry.WithMaxSize(8192),
			ipregistry.WithTTL(10*time.Minute),
		)),
	)
	defer client.Close()

	_, _ = client.Lookup(context.Background(), "8.8.8.8")
}
