package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"sync"
)

type Server struct {
	URL         *url.URL
	ActiveConns int
	mu          sync.Mutex
}

type LoadBalancer struct {
	servers []*Server
}

func NewLoadBalancer(serverURLs []string) *LoadBalancer {
	servers := []*Server{}
	for _, addr := range serverURLs {
		parsedURL, err := url.Parse(addr)
		if err != nil {
			log.Fatalf("Invalid server URL: %s", addr)
		}
		servers = append(servers, &Server{URL: parsedURL})
	}
	return &LoadBalancer{servers: servers}
}

func (lb *LoadBalancer) getLeastConnectionsServer() *Server {
	var selectedServer *Server
	minConns := int(^uint(0) >> 1) // Set to max int value initially

	for _, server := range lb.servers {
		server.mu.Lock()
		if server.ActiveConns < minConns {
			minConns = server.ActiveConns
			selectedServer = server
		}
		server.mu.Unlock()
	}
	return selectedServer
}

func (lb *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	server := lb.getLeastConnectionsServer()
	if server == nil {
		http.Error(w, "No available servers", http.StatusServiceUnavailable)
		return
	}

	// Increment active connections
	server.mu.Lock()
	server.ActiveConns++
	server.mu.Unlock()

	// Decrement active connections after request is handled
	defer func() {
		server.mu.Lock()
		server.ActiveConns--
		server.mu.Unlock()
	}()

	proxyURL := fmt.Sprintf("%s%s", server.URL.String(), r.URL.Path)
	resp, err := http.Get(proxyURL)
	if err != nil {
		http.Error(w, "Backend server error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Forward response to client
	w.WriteHeader(resp.StatusCode)
	_, err = w.Write([]byte(fmt.Sprintf("Response from %s", server.URL.String())))
	if err != nil {
		http.Error(w, "Failed to write response", http.StatusInternalServerError)
	}
}
