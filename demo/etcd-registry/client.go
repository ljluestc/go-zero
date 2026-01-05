package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zeromicro/go-zero/core/discov"
)

func main() {
	// Create a subscriber to discover the demo-api service from etcd
	subscriber, err := discov.NewSubscriber(
		[]string{"localhost:2379"}, // etcd endpoints
		"demo-api",                // service key to discover
	)
	if err != nil {
		fmt.Printf("Failed to create subscriber: %v\n", err)
		return
	}

	// Give some time for discovery
	time.Sleep(2 * time.Second)

	// Get the list of available endpoints
	endpoints := subscriber.Values()
	if len(endpoints) == 0 {
		fmt.Println("No endpoints found for demo-api service")
		return
	}

	fmt.Printf("Found %d endpoints for demo-api service:\n", len(endpoints))
	for i, endpoint := range endpoints {
		fmt.Printf("  %d. %s\n", i+1, endpoint)
	}

	// Call the API on the first available endpoint
	endpoint := endpoints[0]
	fmt.Printf("\nCalling API at endpoint: %s\n", endpoint)

	// Make HTTP request to the discovered service
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "http://"+endpoint+"/api/v1/hello", nil)
	if err != nil {
		fmt.Printf("Failed to create request: %v\n", err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to call API: %v\n", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Failed to read response: %v\n", err)
		return
	}

	fmt.Printf("Response Status: %s\n", resp.Status)
	fmt.Printf("Response Body: %s\n", string(body))
}