package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MaxURLLength      = 2048
	MaxURLsPerRequest = 20
	DefaultLimit      = 20
	MaxLimit          = 100
)

type ProductCatalog struct {
	mu            sync.RWMutex
	products      map[string]*Product // id -> product
	skuIndex      map[string]string   // sku -> id
	nextID        int
	productOrder  []string // maintains insertion order for pagination
}

func NewProductCatalog() *ProductCatalog {
	return &ProductCatalog{
		products:     make(map[string]*Product),
		skuIndex:     make(map[string]string),
		nextID:       1,
		productOrder: []string{},
	}
}

// Validation helpers

func isValidURL(urlStr string) bool {
	if len(urlStr) > MaxURLLength {
		return false
	}
	
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}
	
	scheme := strings.ToLower(parsedURL.Scheme)
	return scheme == "http" || scheme == "https"
}

func validateURLs(urls []string, maxCount int) error {
	if len(urls) > maxCount {
		return fmt.Errorf("maximum %d URLs allowed per request", maxCount)
	}
	
	for _, urlStr := range urls {
		if !isValidURL(urlStr) {
			return fmt.Errorf("invalid URL: %s (must be http/https and max %d characters)", urlStr, MaxURLLength)
		}
	}
	
	return nil
}

// CreateProduct creates a new product
func (pc *ProductCatalog) CreateProduct(req ProductCreateRequest) (*Product, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	// Validate name and SKU
	if strings.TrimSpace(req.Name) == "" {
		return nil, fmt.Errorf("name cannot be empty")
	}
	
	if strings.TrimSpace(req.SKU) == "" {
		return nil, fmt.Errorf("sku cannot be empty")
	}
	
	// Check SKU uniqueness
	if _, exists := pc.skuIndex[req.SKU]; exists {
		return nil, fmt.Errorf("sku already exists")
	}
	
	// Validate URLs
	if err := validateURLs(req.ImageURLs, MaxURLsPerRequest); err != nil {
		return nil, err
	}
	
	if err := validateURLs(req.VideoURLs, MaxURLsPerRequest); err != nil {
		return nil, err
	}
	
	// Create product
	id := fmt.Sprintf("%d", pc.nextID)
	pc.nextID++
	
	product := &Product{
		ID:        id,
		Name:      req.Name,
		SKU:       req.SKU,
		ImageURLs: req.ImageURLs,
		VideoURLs: req.VideoURLs,
		CreatedAt: time.Now(),
	}
	
	if product.ImageURLs == nil {
		product.ImageURLs = []string{}
	}
	if product.VideoURLs == nil {
		product.VideoURLs = []string{}
	}
	
	pc.products[id] = product
	pc.skuIndex[req.SKU] = id
	pc.productOrder = append(pc.productOrder, id)
	
	return product, nil
}

// ListProducts returns paginated products without full media arrays
func (pc *ProductCatalog) ListProducts(limit, offset int) ProductListResponse {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	total := len(pc.productOrder)
	
	// Calculate pagination
	if offset >= total {
		return ProductListResponse{
			Products: []ProductListItem{},
			Total:    total,
			Limit:    limit,
			Offset:   offset,
			HasMore:  false,
		}
	}
	
	end := offset + limit
	if end > total {
		end = total
	}
	
	// Build list items (without full media arrays)
	items := []ProductListItem{}
	for i := offset; i < end; i++ {
		id := pc.productOrder[i]
		product := pc.products[id]
		
		item := ProductListItem{
			ID:         product.ID,
			Name:       product.Name,
			SKU:        product.SKU,
			ImageCount: len(product.ImageURLs),
			VideoCount: len(product.VideoURLs),
			CreatedAt:  product.CreatedAt,
		}
		
		// Add thumbnail if available
		if len(product.ImageURLs) > 0 {
			item.ThumbnailURL = product.ImageURLs[0]
		}
		
		items = append(items, item)
	}
	
	return ProductListResponse{
		Products: items,
		Total:    total,
		Limit:    limit,
		Offset:   offset,
		HasMore:  end < total,
	}
}

// GetProduct returns full product details including all media
func (pc *ProductCatalog) GetProduct(id string) (*Product, error) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	
	product, exists := pc.products[id]
	if !exists {
		return nil, fmt.Errorf("product not found")
	}
	
	return product, nil
}

// AppendMedia adds new media URLs to an existing product
func (pc *ProductCatalog) AppendMedia(id string, req MediaAppendRequest) (*Product, error) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	
	product, exists := pc.products[id]
	if !exists {
		return nil, fmt.Errorf("product not found")
	}
	
	// Validate at least one array has URLs
	if len(req.ImageURLs) == 0 && len(req.VideoURLs) == 0 {
		return nil, fmt.Errorf("at least one of image_urls or video_urls must be provided")
	}
	
	// Validate URLs
	if err := validateURLs(req.ImageURLs, MaxURLsPerRequest); err != nil {
		return nil, err
	}
	
	if err := validateURLs(req.VideoURLs, MaxURLsPerRequest); err != nil {
		return nil, err
	}
	
	// Append URLs
	product.ImageURLs = append(product.ImageURLs, req.ImageURLs...)
	product.VideoURLs = append(product.VideoURLs, req.VideoURLs...)
	
	return product, nil
}

// HTTP Handlers for Part 2

func (pc *ProductCatalog) HandleCreateProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req ProductCreateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON"})
		return
	}
	
	product, err := pc.CreateProduct(req)
	if err != nil {
		if strings.Contains(err.Error(), "already exists") {
			w.WriteHeader(http.StatusConflict)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
	
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(product)
}

func (pc *ProductCatalog) HandleListProducts(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Parse pagination parameters
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	limit := DefaultLimit
	offset := 0
	
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 {
			limit = parsed
			if limit > MaxLimit {
				limit = MaxLimit
			}
		}
	}
	
	if offsetStr != "" {
		if parsed, err := strconv.Atoi(offsetStr); err == nil && parsed >= 0 {
			offset = parsed
		}
	}
	
	response := pc.ListProducts(limit, offset)
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (pc *ProductCatalog) HandleGetProduct(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/products/")
	id := strings.Split(path, "/")[0]
	
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Product ID required"})
		return
	}
	
	product, err := pc.GetProduct(id)
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Product not found"})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

func (pc *ProductCatalog) HandleAppendMedia(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	// Extract ID from path
	path := strings.TrimPrefix(r.URL.Path, "/products/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 || parts[1] != "media" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid path"})
		return
	}
	id := parts[0]
	
	var req MediaAppendRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON"})
		return
	}
	
	product, err := pc.AppendMedia(id, req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			w.WriteHeader(http.StatusNotFound)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
		json.NewEncoder(w).Encode(ErrorResponse{Error: err.Error()})
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}
