package test

import (
	"net/http"
	"testing"
)

// TestHTTPCall tests the HTTP server by sending a request to open and then send.
func TestHTTPCall(t *testing.T) {
	// Define the server URL
	baseURL := "http://localhost:8080"

	// Send a GET request to the open endpoint
	openResp, err := http.Get(baseURL + "/open")
	if err != nil {
		t.Fatalf("Failed to send GET request to /open: %v", err)
	}
	defer openResp.Body.Close()

	// Check if the open response status is OK
	if openResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK from /open, got %v", openResp.Status)
	}

	// Send a GET request to the send endpoint
	sendResp, err := http.Get(baseURL + "/send")
	if err != nil {
		t.Fatalf("Failed to send GET request to /send: %v", err)
	}
	defer sendResp.Body.Close()

	// Check if the send response status is OK
	if sendResp.StatusCode != http.StatusOK {
		t.Fatalf("Expected status OK from /send, got %v", sendResp.Status)
	}

	// Add more assertions based on expected response
}
