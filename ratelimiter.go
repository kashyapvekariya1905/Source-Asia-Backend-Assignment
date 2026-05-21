package main

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

const (
	// Rate limit: 5 requests per user per minute
	MaxRequestsPerMinute = 5
	WindowDuration       = 1 * time.Minute
)

type RequestRecord struct {
	Timestamp time.Time
}

type RateLimiter struct {
	mu               sync.RWMutex
	userRequests     map[string][]RequestRecord // user_id -> list of request timestamps
	rejectedRequests map[string]int             // user_id -> cumulative rejected count
}

func NewRateLimiter() *RateLimiter {
	rl := &RateLimiter{
		userRequests:     make(map[string][]RequestRecord),
		rejectedRequests: make(map[string]int),
	}
	
	// Start background cleanup of old records
	go rl.cleanupOldRecords()
	
	return rl
}

// cleanupOldRecords runs periodically to remove expired request records
func (rl *RateLimiter) cleanupOldRecords() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for userID, records := range rl.userRequests {
			validRecords := []RequestRecord{}
			for _, record := range records {
				if now.Sub(record.Timestamp) < WindowDuration {
					validRecords = append(validRecords, record)
				}
			}
			if len(validRecords) > 0 {
				rl.userRequests[userID] = validRecords
			} else {
				delete(rl.userRequests, userID)
			}
		}
		rl.mu.Unlock()
	}
}

// Allow checks if a request should be allowed and records it if so
func (rl *RateLimiter) Allow(userID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	
	now := time.Now()
	
	// Get current valid requests (within the window)
	validRecords := []RequestRecord{}
	for _, record := range rl.userRequests[userID] {
		if now.Sub(record.Timestamp) < WindowDuration {
			validRecords = append(validRecords, record)
		}
	}
	
	// Check if limit exceeded
	if len(validRecords) >= MaxRequestsPerMinute {
		// Rate limit exceeded - increment rejection counter
		rl.rejectedRequests[userID]++
		// Update with cleaned records
		rl.userRequests[userID] = validRecords
		return false
	}
	
	// Allow request - add new record
	validRecords = append(validRecords, RequestRecord{Timestamp: now})
	rl.userRequests[userID] = validRecords
	
	return true
}

// GetStats returns current statistics for all users
func (rl *RateLimiter) GetStats() map[string]UserStats {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	
	stats := make(map[string]UserStats)
	now := time.Now()
	
	// Count accepted requests in current window for each user
	for userID, records := range rl.userRequests {
		acceptedCount := 0
		for _, record := range records {
			if now.Sub(record.Timestamp) < WindowDuration {
				acceptedCount++
			}
		}
		
		stats[userID] = UserStats{
			AcceptedInCurrentWindow: acceptedCount,
			RejectedCumulative:      rl.rejectedRequests[userID],
		}
	}
	
	// Include users who only have rejections
	for userID, rejectedCount := range rl.rejectedRequests {
		if _, exists := stats[userID]; !exists {
			stats[userID] = UserStats{
				AcceptedInCurrentWindow: 0,
				RejectedCumulative:      rejectedCount,
			}
		}
	}
	
	return stats
}

// HTTP Handlers for Part 1

func (rl *RateLimiter) HandleRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	var req RequestPayload
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "Invalid JSON"})
		return
	}
	
	// Validate user_id
	if req.UserID == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "user_id is required and cannot be empty"})
		return
	}
	
	// Validate payload exists
	if req.Payload == nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(ErrorResponse{Error: "payload is required"})
		return
	}
	
	// Check rate limit
	if !rl.Allow(req.UserID) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(ErrorResponse{
			Error: "Rate limit exceeded. Maximum 5 requests per minute per user.",
		})
		return
	}
	
	// Request accepted
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(RequestResponse{
		Message: "Request accepted",
		UserID:  req.UserID,
	})
}

func (rl *RateLimiter) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	
	stats := rl.GetStats()
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(StatsResponse{Stats: stats})
}
