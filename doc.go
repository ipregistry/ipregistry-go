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

// Package ipregistry is the official Go client library for the Ipregistry
// (https://ipregistry.co) IP geolocation and threat data API.
//
// It lets you look up your own IP address or arbitrary ones. Responses return
// multiple data points including carrier, company, connection, currency,
// location, time zone, and threat information, and it can also parse raw
// User-Agent strings.
//
// # Getting started
//
// You need an Ipregistry API key, which you can get along with a generous free
// tier by signing up at https://ipregistry.co.
//
//	client := ipregistry.New("YOUR_API_KEY")
//	defer client.Close()
//
//	info, err := client.Lookup(context.Background(), "8.8.8.8")
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(info.Location.Country.Name)
//
// # Origin lookup
//
// To retrieve data for the IP address the request originates from, use
// [Client.LookupOrigin], which additionally returns parsed User-Agent data:
//
//	origin, err := client.LookupOrigin(context.Background())
//
// # Batch lookups
//
// [Client.LookupBatch] resolves many IP addresses in a single request. Each
// entry may independently succeed or fail (for example on an invalid address),
// so results are inspected element by element:
//
//	list, err := client.LookupBatch(ctx, []string{"8.8.8.8", "1.1.1.1"})
//	if err != nil {
//		log.Fatal(err)
//	}
//	for info, err := range list.All() {
//		if err != nil {
//			log.Println("entry failed:", err)
//			continue
//		}
//		fmt.Println(info.Location.Country.Name)
//	}
//
// # Errors
//
// API-level failures are reported as [*APIError], which carries both the raw
// Ipregistry error code and a typed [ErrorCode]. Transport and decoding
// failures are reported as [*ClientError]. Both can be matched with errors.As.
//
// # Concurrency
//
// A [Client] is safe for concurrent use by multiple goroutines. Because every
// method takes a context.Context, cancellation and timeouts compose naturally
// with Go's concurrency primitives; there is no separate asynchronous API.
package ipregistry
