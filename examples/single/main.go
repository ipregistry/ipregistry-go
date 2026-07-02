// Command single performs a single IP address lookup.
//
// Usage:
//
//	IPREGISTRY_API_KEY=YOUR_API_KEY go run ./examples/single [ip]
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	ipregistry "github.com/ipregistry/ipregistry-go"
)

func main() {
	apiKey := os.Getenv("IPREGISTRY_API_KEY")
	if apiKey == "" {
		log.Fatal("set IPREGISTRY_API_KEY")
	}

	ip := "8.8.8.8"
	if len(os.Args) > 1 {
		ip = os.Args[1]
	}

	client := ipregistry.New(apiKey)
	defer client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	info, err := client.Lookup(ctx, ip)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("IP:       %s (%s)\n", info.IP, info.Type)
	fmt.Printf("Country:  %s\n", info.Location.Country.Name)
	fmt.Printf("City:     %s\n", info.Location.City)
	if info.Connection.ASN != nil {
		fmt.Printf("ASN:      %d (%s)\n", *info.Connection.ASN, info.Connection.Organization)
	}
	fmt.Printf("Currency: %s\n", info.Currency.Code)
	fmt.Printf("Timezone: %s\n", info.TimeZone.ID)
	fmt.Printf("Threat:   %t\n", info.Security.IsThreat)
}
