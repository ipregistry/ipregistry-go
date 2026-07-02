// Command origin looks up the IP address the request originates from and prints
// the parsed User-Agent data returned alongside it.
//
// Usage:
//
//	IPREGISTRY_API_KEY=YOUR_API_KEY go run ./examples/origin
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	ipregistry "github.com/ipregistry/ipregistry-go"
)

func main() {
	apiKey := os.Getenv("IPREGISTRY_API_KEY")
	if apiKey == "" {
		log.Fatal("set IPREGISTRY_API_KEY")
	}

	client := ipregistry.New(apiKey)
	defer client.Close()

	origin, err := client.LookupOrigin(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Your IP:  %s\n", origin.IP)
	fmt.Printf("Country:  %s\n", origin.Location.Country.Name)
	if origin.UserAgent != nil {
		fmt.Printf("Browser:  %s %s\n", origin.UserAgent.Name, origin.UserAgent.Version)
		fmt.Printf("OS:       %s\n", origin.UserAgent.OperatingSystem.Name)
	}
}
