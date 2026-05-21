package main

import "time"

// Part 1: Rate Limiter Models

type RequestPayload struct {
	UserID  string      `json:"user_id"`
	Payload interface{} `json:"payload"`
}

type RequestResponse struct {
	Message string `json:"message"`
	UserID  string `json:"user_id"`
}

type UserStats struct {
	AcceptedInCurrentWindow int `json:"accepted_in_current_window"`
	RejectedCumulative      int `json:"rejected_cumulative"`
}

type StatsResponse struct {
	Stats map[string]UserStats `json:"stats"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// Part 2: Product Catalog Models

type Product struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	SKU       string    `json:"sku"`
	ImageURLs []string  `json:"image_urls"`
	VideoURLs []string  `json:"video_urls"`
	CreatedAt time.Time `json:"created_at"`
}

type ProductCreateRequest struct {
	Name      string   `json:"name"`
	SKU       string   `json:"sku"`
	ImageURLs []string `json:"image_urls,omitempty"`
	VideoURLs []string `json:"video_urls,omitempty"`
}

type ProductListItem struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	SKU          string    `json:"sku"`
	ImageCount   int       `json:"image_count"`
	VideoCount   int       `json:"video_count"`
	ThumbnailURL string    `json:"thumbnail_url,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type ProductListResponse struct {
	Products   []ProductListItem `json:"products"`
	Total      int               `json:"total"`
	Limit      int               `json:"limit"`
	Offset     int               `json:"offset"`
	HasMore    bool              `json:"has_more"`
}

type MediaAppendRequest struct {
	ImageURLs []string `json:"image_urls,omitempty"`
	VideoURLs []string `json:"video_urls,omitempty"`
}
