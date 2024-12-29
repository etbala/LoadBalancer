package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func startMockServer(response string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, response)
	}))
}

func TestLoadBalancer(t *testing.T) {
	// Mock backend servers
	server1 := startMockServer("Response from server 1")
	defer server1.Close()

	server2 := startMockServer("Response from server 2")
	defer server2.Close()

	server3 := startMockServer("Response from server 3")
	defer server3.Close()

	// Create load balancer
	servers := []string{server1.URL, server2.URL, server3.URL}
	lb := NewLoadBalancer(servers)

	responseCounts := map[string]int{}
	mu := sync.Mutex{} // Mutex to protect the map during concurrent writes

	// Simulate concurrent requests
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			w := httptest.NewRecorder()

			lb.ServeHTTP(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			// Record which server responded
			mu.Lock()
			responseCounts[string(body)]++
			mu.Unlock()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Unexpected status code: got %v, want %v", resp.StatusCode, http.StatusOK)
			}
		}()
	}

	// Wait for all Goroutines to finish
	wg.Wait()

	// Print response counts
	for server, count := range responseCounts {
		fmt.Printf("%s handled %d requests\n", server, count)
	}

	if len(responseCounts) < len(servers) {
		t.Errorf("Not all servers were utilized: %v", responseCounts)
	}
}
