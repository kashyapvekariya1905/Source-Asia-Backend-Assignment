package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

func main() {
	// Initialize Part 1: Rate Limiter
	rateLimiter := NewRateLimiter()
	
	// Initialize Part 2: Product Catalog
	productCatalog := NewProductCatalog()
	
	// Part 1 endpoints
	http.HandleFunc("/request", rateLimiter.HandleRequest)
	http.HandleFunc("/stats", rateLimiter.HandleStats)
	
	// Part 2 endpoints
	http.HandleFunc("/products", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			productCatalog.HandleCreateProduct(w, r)
		} else if r.Method == http.MethodGet {
			productCatalog.HandleListProducts(w, r)
		} else {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	})
	
	http.HandleFunc("/products/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		
		// Check if this is a media append request
		if strings.Contains(path, "/media") {
			productCatalog.HandleAppendMedia(w, r)
		} else {
			// This is a get single product request
			productCatalog.HandleGetProduct(w, r)
		}
	})
	
	// Health check endpoint
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	
	port := 8080
	fmt.Printf("Server starting on port %d...\n", port)
	fmt.Println("Part 1 endpoints:")
	fmt.Println("  POST /request")
	fmt.Println("  GET  /stats")
	fmt.Println("\nPart 2 endpoints:")
	fmt.Println("  POST /products")
	fmt.Println("  GET  /products")
	fmt.Println("  GET  /products/{id}")
	fmt.Println("  POST /products/{id}/media")
	fmt.Println("\nHealthcheck:")
	fmt.Println("  GET  /health")
	
	if err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil); err != nil {
		log.Fatal(err)
	}
}
