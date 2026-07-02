// Command batch performs a batch lookup with in-memory caching enabled.
//
// Usage:
//
//	IPREGISTRY_API_KEY=YOUR_API_KEY go run ./examples/batch
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	ipregistry "github.com/ipregistry/ipregistry-go"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	apiKey := os.Getenv("IPREGISTRY_API_KEY")
	if apiKey == "" {
		return errors.New("set IPREGISTRY_API_KEY")
	}

	client := ipregistry.New(apiKey,
		ipregistry.WithCache(ipregistry.NewInMemoryCache()),
	)
	defer client.Close()

	ips := []string{"73.2.2.2", "8.8.8.8", "2001:67c:2e8:22::c100:68b", "not-an-ip"}

	list, err := client.LookupBatch(context.Background(), ips)
	if err != nil {
		return err
	}

	for i, ip := range ips {
		info, err := list.At(i)
		if err != nil {
			fmt.Printf("%-40s error: %v\n", ip, err)
			continue
		}
		fmt.Printf("%-40s %s\n", ip, info.Location.Country.Name)
	}
	return nil
}
