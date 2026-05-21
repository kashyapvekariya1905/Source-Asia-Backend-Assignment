# How to Run - Source Asia Backend Assignment

This guide will walk you through running and testing the backend assignment step by step.

## Prerequisites

1. **Install Go** (version 1.21 or higher)
   - Download from: https://go.dev/dl/
   - Verify installation: `go version`

2. **Install curl** (for testing)
   - Usually pre-installed on Linux/Mac
   - Windows: Download from https://curl.se/windows/

3. **Optional: Install jq** (for pretty JSON output)
   - Linux: `sudo apt-get install jq`
   - Mac: `brew install jq`
   - Windows: Download from https://stedolan.github.io/jq/

---

## Step 1: Extract the Project

```bash
# Extract the zip file
cd source-asia-backend
```

---

## Step 2: Start the Server

### Option A: Run directly (recommended for testing)

```bash
go run .
```

### Option B: Build and run

```bash
go build -o server
./server
```

You should see:
```
Server starting on port 8080...
Part 1 endpoints:
  POST /request
  GET  /stats

Part 2 endpoints:
  POST /products
  GET  /products
  GET  /products/{id}
  POST /products/{id}/media

Healthcheck:
  GET  /health
```

**Keep this terminal open.** The server must be running for all tests.

---

## Step 3: Quick Smoke Test

Run all at once(in different terminal):
```bash
chmod +x test_all.sh
./test_all.sh
```
Open a **new terminal** and test the health endpoint:

```bash
curl http://localhost:8080/health
```

Expected output: `OK`

---

## Step 4: Test Part 1 (Rate Limiter)

### Test 4.1: Send valid requests (first 5 should succeed)

```bash
# Request 1
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","payload":{"action":"test","count":1}}'

# Request 2
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","payload":{"action":"test","count":2}}'

# Request 3
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","payload":{"action":"test","count":3}}'

# Request 4
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","payload":{"action":"test","count":4}}'

# Request 5
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","payload":{"action":"test","count":5}}'
```

**Expected**: Each returns `201 Created` with:
```json
{"message":"Request accepted","user_id":"alice"}
```

### Test 4.2: Test rate limit (6th request should fail)

```bash
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","payload":{"action":"test","count":6}}'
```

**Expected**: `429 Too Many Requests` with:
```json
{"error":"Rate limit exceeded. Maximum 5 requests per minute per user."}
```

### Test 4.3: Check statistics

```bash
curl http://localhost:8080/stats
```

**Expected**:
```json
{
  "stats": {
    "alice": {
      "accepted_in_current_window": 5,
      "rejected_cumulative": 1
    }
  }
}
```

### Test 4.4: Test invalid input

```bash
# Empty user_id (should return 400)
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"","payload":{"test":1}}'

# Missing user_id (should return 400)
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"payload":{"test":1}}'
```

### Test 4.5: Test different users (isolation)

```bash
# Bob should be able to make requests (separate from Alice's limit)
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"bob","payload":{"test":1}}'
```

**Expected**: `201 Created` (bob has his own rate limit)

---

## Step 5: Test Part 2 (Product Catalog)

### Test 5.1: Create a product

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Ergonomic Keyboard",
    "sku": "KB-2024-001",
    "image_urls": [
      "https://cdn.example.com/kb/front.jpg",
      "https://cdn.example.com/kb/side.jpg",
      "https://cdn.example.com/kb/top.jpg"
    ],
    "video_urls": [
      "https://cdn.example.com/kb/unboxing.mp4"
    ]
  }'
```

**Expected**: `201 Created` with full product details including an `id` field.

### Test 5.2: List products (note: no full media arrays)

```bash
curl "http://localhost:8080/products?limit=10&offset=0"
```

**Expected**: Response with `ProductListItem` objects that have:
- `image_count` and `video_count` (NOT full arrays)
- Pagination metadata: `total`, `limit`, `offset`, `has_more`

### Test 5.3: Get product detail

```bash
# Replace {id} with the actual ID from the create response
curl http://localhost:8080/products/1
```

**Expected**: Full product with complete `image_urls` and `video_urls` arrays.

### Test 5.4: Append media to product

```bash
curl -X POST http://localhost:8080/products/1/media \
  -H "Content-Type: application/json" \
  -d '{
    "image_urls": ["https://cdn.example.com/kb/detail.jpg"],
    "video_urls": ["https://cdn.example.com/kb/review.mp4"]
  }'
```

**Expected**: `200 OK` with updated product (now 4 images, 2 videos).

### Test 5.5: Test duplicate SKU

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Duplicate Test","sku":"KB-2024-001"}'
```

**Expected**: `409 Conflict` with error message.

### Test 5.6: Test invalid URL

```bash
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Product",
    "sku": "TEST-001",
    "image_urls": ["not-a-valid-url"]
  }'
```

**Expected**: `400 Bad Request` with validation error.

### Test 5.7: Test non-existent product

```bash
curl http://localhost:8080/products/99999
```

**Expected**: `404 Not Found`

---

## Step 6: Run Automated Test Suite

We've included a comprehensive test script:

```bash
chmod +x test_all.sh
./test_all.sh
```

This will run through all the test cases automatically and show you pass/fail results.

---

## Step 7: Performance Testing (Optional)

### Create 1000 products with 10 images each:

```bash
chmod +x seed_products.sh
./seed_products.sh
```

This takes a few minutes but demonstrates that `GET /products` stays fast even with:
- 1,000 products
- 10,000 total image URLs stored

After seeding, test the list endpoint:

```bash
# Should be fast (< 100ms) because it doesn't load all 10,000 URLs
time curl -s "http://localhost:8080/products?limit=20" | jq '.products | length'
```

### Test pagination:

```bash
# First page
curl "http://localhost:8080/products?limit=20&offset=0" | jq

# Fifth page
curl "http://localhost:8080/products?limit=20&offset=80" | jq

# Last page
curl "http://localhost:8080/products?limit=20&offset=980" | jq
```

---

## Step 8: Concurrent Request Testing (Advanced)

### Test concurrent rate limiting:

If you have GNU Parallel installed:

```bash
# Send 10 concurrent requests for the same user
seq 10 | parallel -j 10 curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"testuser","payload":{"request":{}}}'

# Check stats - should show exactly 5 accepted, 5 rejected
curl http://localhost:8080/stats | jq '.stats.testuser'
```

Without parallel, use separate terminal windows or a simple bash loop:

```bash
# In terminal 1-5: Run these simultaneously (hit enter at the same time)
curl -X POST http://localhost:8080/request -H "Content-Type: application/json" -d '{"user_id":"concurrent","payload":{"n":1}}'
```

---

## Common Issues & Troubleshooting

### Issue: "connection refused"
**Solution**: Make sure the server is running in another terminal (`go run .`)

### Issue: "command not found: go"
**Solution**: Install Go from https://go.dev/dl/

### Issue: Rate limit not working as expected
**Solution**: Wait 60 seconds for the window to reset, or restart the server (state is in-memory)

### Issue: curl showing HTML instead of JSON
**Solution**: Check the URL path - you might be hitting a 404. The server only returns JSON for valid endpoints.

### Issue: jq not working
**Solution**: Either install jq or remove `| jq` from commands to see raw JSON

---

## Stopping the Server

In the terminal where the server is running, press `Ctrl+C` to stop it gracefully.

---

## API Reference Quick Guide

### Part 1 Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| POST | /request | Submit a request (rate limited) |
| GET | /stats | Get rate limit statistics |

### Part 2 Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| POST | /products | Create a new product |
| GET | /products | List products (paginated, counts only) |
| GET | /products/{id} | Get full product details |
| POST | /products/{id}/media | Append media URLs to product |

---

## Next Steps

1.  Run the server
2.  Test Part 1 (rate limiter)
3.  Test Part 2 (product catalog)
4.  Run automated tests (`./test_all.sh`)
5.  (Optional) Run performance tests (`./seed_products.sh`)
6.  Review the README.md for design decisions and production considerations

---

## Questions?

If you encounter any issues or have questions about the implementation:

1. Check the README.md for detailed API documentation
2. Review the code comments in the source files
3. Check server logs in the terminal where `go run .` is running

All requirements from the assignment have been implemented and tested. The codebase is ready for review!
