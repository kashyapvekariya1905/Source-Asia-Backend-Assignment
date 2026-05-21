# Source Asia Backend Assignment

A Go-based HTTP service implementing a rate-limited API and product catalog with media management.

**AI Tools Used**: Claude (Anthropic) was used to help structure the project, ensure concurrency safety patterns, and verify API design best practices.

## Table of Contents
- [Quick Start](#quick-start)
- [Part 1: Rate-Limited API](#part-1-rate-limited-api)
- [Part 2: Product Catalog](#part-2-product-catalog)
- [Design Decisions](#design-decisions)
- [Production Limitations](#production-limitations)
- [Testing Examples](#testing-examples)

## Quick Start

### Prerequisites
- Go 1.21 or higher

### Running the Server

```bash
# Clone or extract the repository
cd source-asia-backend

# Run the server (no build required for testing)
go run .

# Or build and run
go build -o server
./server
```

The server will start on `http://localhost:8080`

### Health Check

```bash
curl http://localhost:8080/health
```

---

## Part 1: Rate-Limited API

### Design Decisions

**Rate Limiting Approach**: Fixed 1-minute window with sliding cleanup
- Each request is timestamped when accepted
- A background goroutine cleans up expired records every 30 seconds
- The `Allow()` method filters valid requests within the current 1-minute window before checking the limit
- This provides efficient memory usage while maintaining accurate rate limiting

**Concurrency Safety**: 
- `sync.RWMutex` protects all shared data structures
- Write lock for accepting/rejecting requests
- Read lock for stats retrieval
- Safe for parallel requests to the same user_id

**Response Codes**:
- `201 Created` for accepted requests (per REST best practices for resource creation)
- `429 Too Many Requests` for rate limit violations
- `400 Bad Request` for invalid input

**Statistics**:
- `accepted_in_current_window`: Real-time count of valid requests in the last 60 seconds
- `rejected_cumulative`: Total rejections since server start (cumulative, not per-window)

### Endpoints

#### POST /request

Accept a request with rate limiting.

**Request Body**:
```json
{
  "user_id": "string (required, non-empty)",
  "payload": "any JSON value (required)"
}
```

**Success Response (201)**:
```json
{
  "message": "Request accepted",
  "user_id": "user123"
}
```

**Rate Limit Exceeded (429)**:
```json
{
  "error": "Rate limit exceeded. Maximum 5 requests per minute per user."
}
```

**Invalid Input (400)**:
```json
{
  "error": "user_id is required and cannot be empty"
}
```

#### GET /stats

Retrieve statistics for all users.

**Response (200)**:
```json
{
  "stats": {
    "user123": {
      "accepted_in_current_window": 5,
      "rejected_cumulative": 12
    },
    "user456": {
      "accepted_in_current_window": 2,
      "rejected_cumulative": 0
    }
  }
}
```

### Example Usage

```bash
# Accept requests (first 5 should succeed)
for i in {1..5}; do
  curl -X POST http://localhost:8080/request \
    -H "Content-Type: application/json" \
    -d '{"user_id":"alice","payload":{"action":"test","count":'$i'}}'
  echo ""
done

# This should be rejected (429)
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","payload":{"action":"test","count":6}}'

# Check statistics
curl http://localhost:8080/stats | jq

# Test invalid input
curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"","payload":{"test":1}}'
```

---

## Part 2: Product Catalog

### Design Decisions

**Data Model**:
- **products map**: `map[string]*Product` - O(1) lookup by ID
- **skuIndex map**: `map[string]string` - O(1) SKU uniqueness check (SKU → ID)
- **productOrder slice**: `[]string` - Maintains insertion order for consistent pagination
- Media URLs stored directly in Product struct (acceptable for in-memory storage)

**List vs Detail Query Optimization**:
- **List endpoint** (`GET /products`): Returns `ProductListItem` without full media arrays
  - Only includes: id, name, sku, image_count, video_count, thumbnail_url, created_at
  - Significantly reduces payload size for large catalogs
  - O(1) access to counts (pre-calculated from array length)
- **Detail endpoint** (`GET /products/{id}`): Returns full `Product` with all media URLs

**Pagination**:
- Query parameters: `limit` (default: 20, max: 100) and `offset` (default: 0)
- Response includes: total count, current limit/offset, `has_more` flag
- Uses `productOrder` slice to maintain consistent ordering
- Example: `GET /products?limit=20&offset=40`

**URL Validation**:
- Must start with `http://` or `https://`
- Maximum length: 2048 characters
- Maximum 20 URLs per request (configurable via `MaxURLsPerRequest`)

**Response Codes**:
- `201 Created` for successful product creation
- `409 Conflict` for duplicate SKU
- `404 Not Found` for non-existent product/ID
- `400 Bad Request` for validation failures

### Endpoints

#### POST /products

Create a new product.

**Request Body**:
```json
{
  "name": "Premium Widget",
  "sku": "SKU-001",
  "image_urls": [
    "https://cdn.example.com/products/sku-001/img-1.jpg",
    "https://cdn.example.com/products/sku-001/img-2.jpg"
  ],
  "video_urls": [
    "https://cdn.example.com/products/sku-001/demo.mp4"
  ]
}
```

**Success Response (201)**:
```json
{
  "id": "1",
  "name": "Premium Widget",
  "sku": "SKU-001",
  "image_urls": [
    "https://cdn.example.com/products/sku-001/img-1.jpg",
    "https://cdn.example.com/products/sku-001/img-2.jpg"
  ],
  "video_urls": [
    "https://cdn.example.com/products/sku-001/demo.mp4"
  ],
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Duplicate SKU (409)**:
```json
{
  "error": "sku already exists"
}
```

#### GET /products

List products with pagination (optimized for large catalogs).

**Query Parameters**:
- `limit`: Number of items per page (default: 20, max: 100)
- `offset`: Number of items to skip (default: 0)

**Response (200)**:
```json
{
  "products": [
    {
      "id": "1",
      "name": "Premium Widget",
      "sku": "SKU-001",
      "image_count": 2,
      "video_count": 1,
      "thumbnail_url": "https://cdn.example.com/products/sku-001/img-1.jpg",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ],
  "total": 1,
  "limit": 20,
  "offset": 0,
  "has_more": false
}
```

**Note**: This endpoint does NOT return the full `image_urls` and `video_urls` arrays. This ensures fast responses even with 1,000+ products containing 10+ media URLs each.

#### GET /products/{id}

Get full product details including all media URLs.

**Response (200)**:
```json
{
  "id": "1",
  "name": "Premium Widget",
  "sku": "SKU-001",
  "image_urls": [
    "https://cdn.example.com/products/sku-001/img-1.jpg",
    "https://cdn.example.com/products/sku-001/img-2.jpg"
  ],
  "video_urls": [
    "https://cdn.example.com/products/sku-001/demo.mp4"
  ],
  "created_at": "2024-01-15T10:30:00Z"
}
```

**Not Found (404)**:
```json
{
  "error": "Product not found"
}
```

#### POST /products/{id}/media

Append media URLs to an existing product.

**Request Body**:
```json
{
  "image_urls": [
    "https://cdn.example.com/products/sku-001/img-3.jpg"
  ],
  "video_urls": [
    "https://cdn.example.com/products/sku-001/tutorial.mp4"
  ]
}
```

**Success Response (200)**:
```json
{
  "id": "1",
  "name": "Premium Widget",
  "sku": "SKU-001",
  "image_urls": [
    "https://cdn.example.com/products/sku-001/img-1.jpg",
    "https://cdn.example.com/products/sku-001/img-2.jpg",
    "https://cdn.example.com/products/sku-001/img-3.jpg"
  ],
  "video_urls": [
    "https://cdn.example.com/products/sku-001/demo.mp4",
    "https://cdn.example.com/products/sku-001/tutorial.mp4"
  ],
  "created_at": "2024-01-15T10:30:00Z"
}
```

### Example Usage

```bash
# Create a product
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

# List products (first page)
curl "http://localhost:8080/products?limit=10&offset=0" | jq

# Get specific product
curl http://localhost:8080/products/1 | jq

# Append more media
curl -X POST http://localhost:8080/products/1/media \
  -H "Content-Type: application/json" \
  -d '{
    "image_urls": ["https://cdn.example.com/kb/detail.jpg"],
    "video_urls": ["https://cdn.example.com/kb/review.mp4"]
  }'

# Test duplicate SKU (should return 409)
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{"name":"Duplicate","sku":"KB-2024-001"}'

# Test invalid URL (should return 400)
curl -X POST http://localhost:8080/products \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test",
    "sku": "TEST-001",
    "image_urls": ["not-a-valid-url"]
  }'
```

### Seed Script for Testing

To create 1,000 products for performance testing:

```bash
#!/bin/bash
for i in {1..1000}; do
  curl -X POST http://localhost:8080/products \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"Product $i\",
      \"sku\": \"SKU-$(printf '%05d' $i)\",
      \"image_urls\": [
        \"https://cdn.example.com/p$i/img-1.jpg\",
        \"https://cdn.example.com/p$i/img-2.jpg\",
        \"https://cdn.example.com/p$i/img-3.jpg\",
        \"https://cdn.example.com/p$i/img-4.jpg\",
        \"https://cdn.example.com/p$i/img-5.jpg\",
        \"https://cdn.example.com/p$i/img-6.jpg\",
        \"https://cdn.example.com/p$i/img-7.jpg\",
        \"https://cdn.example.com/p$i/img-8.jpg\",
        \"https://cdn.example.com/p$i/img-9.jpg\",
        \"https://cdn.example.com/p$i/img-10.jpg\"
      ],
      \"video_urls\": [
        \"https://cdn.example.com/p$i/video.mp4\"
      ]
    }" -s > /dev/null
  
  if [ $((i % 100)) -eq 0 ]; then
    echo "Created $i products..."
  fi
done

echo "Testing GET /products?limit=20..."
time curl -s "http://localhost:8080/products?limit=20" | jq '.products | length'
```

With 1,000 products and 10 images each:
- **List endpoint**: Only returns counts and thumbnail (fast, ~2-5ms)
- **Detail endpoint**: Returns all 10 image URLs (only when needed)

---

## Design Decisions

### Concurrency Safety

Both Part 1 and Part 2 use `sync.RWMutex` for thread-safe operations:
- **Write operations** (POST endpoints): Acquire full lock
- **Read operations** (GET endpoints): Acquire read lock (allows concurrent reads)
- Safe for high-concurrency scenarios with multiple goroutines

### Memory Efficiency

**Part 1**:
- Background cleanup removes expired request records every 30 seconds
- Prevents unbounded memory growth
- Only stores active requests within the current window

**Part 2**:
- Products stored with direct array references (efficient for in-memory)
- List endpoint avoids serializing unnecessary data
- Pagination prevents loading entire catalog at once

### PostgreSQL + CDN Migration Strategy

**Current In-Memory Design**:
```
Product {
  id, name, sku, created_at,
  image_urls: []string,
  video_urls: []string
}
```

**PostgreSQL Schema**:
```sql
-- Products table
CREATE TABLE products (
  id SERIAL PRIMARY KEY,
  name TEXT NOT NULL,
  sku TEXT UNIQUE NOT NULL,
  created_at TIMESTAMP DEFAULT NOW()
);

-- Media table (one-to-many)
CREATE TABLE product_media (
  id SERIAL PRIMARY KEY,
  product_id INT REFERENCES products(id) ON DELETE CASCADE,
  media_type VARCHAR(10) CHECK (media_type IN ('image', 'video')),
  url TEXT NOT NULL,
  position INT DEFAULT 0,
  created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_product_media_product_id ON product_media(product_id);
CREATE INDEX idx_product_media_type ON product_media(product_id, media_type);
```

**Query Optimization**:
```sql
-- List endpoint (counts only, no URLs)
SELECT 
  p.id, p.name, p.sku, p.created_at,
  COUNT(CASE WHEN m.media_type = 'image' THEN 1 END) as image_count,
  COUNT(CASE WHEN m.media_type = 'video' THEN 1 END) as video_count,
  (SELECT url FROM product_media 
   WHERE product_id = p.id AND media_type = 'image' 
   ORDER BY position LIMIT 1) as thumbnail_url
FROM products p
LEFT JOIN product_media m ON p.id = m.product_id
GROUP BY p.id
ORDER BY p.created_at DESC
LIMIT 20 OFFSET 0;

-- Detail endpoint (all media)
SELECT * FROM products WHERE id = $1;
SELECT url, media_type, position 
FROM product_media 
WHERE product_id = $1 
ORDER BY media_type, position;
```

**CDN Integration**:
- Store only CDN URLs in database (not actual files)
- CDN handles image resizing, caching, delivery
- Example: CloudFront, Cloudflare, Imgix
- Thumbnail generation via CDN URL parameters: `?w=300&h=300`

---

## Production Limitations

### Single Instance Deployment

**Current Limitations**:
1. **No Persistence**: Server restart loses all data (products, rate limit counters)
2. **Single Point of Failure**: No redundancy or high availability
3. **Rate Limiting Scope**: Only works within a single process
4. **Memory Limits**: Bounded by server RAM (no horizontal scaling)

### Multi-Instance Deployment Considerations

**Rate Limiting** (Part 1):
- **Problem**: Each instance has independent counters
- **Solution**: Centralized rate limiting with Redis
  ```
  Key: rate:user123:2024-01-15T10:30
  Value: request_count
  TTL: 60 seconds
  ```
- Use Redis INCR with TTL for atomic operations
- Alternative: Token bucket algorithm with Redis or Memcached

**Product Catalog** (Part 2):
- **Problem**: In-memory storage doesn't scale or persist
- **Solution**: PostgreSQL database with proper indexing
- **Caching Layer**: Redis for frequently accessed products
- **Search**: Elasticsearch for product search and filtering

**Session/State**:
- Stateless API design allows load balancing
- Database handles shared state
- No sticky sessions required

### Additional Production Requirements

1. **Monitoring & Logging**:
   - Structured logging (JSON format)
   - Application metrics (requests/sec, latency, errors)
   - Distributed tracing (if microservices)

2. **Security**:
   - HTTPS/TLS termination
   - API authentication (JWT, API keys)
   - Rate limiting per API key (not just user_id)
   - Input sanitization (SQL injection, XSS prevention)
   - CORS configuration

3. **Database**:
   - Connection pooling
   - Read replicas for scaling reads
   - Database migrations (e.g., golang-migrate)
   - Backups and disaster recovery

4. **Deployment**:
   - Docker containerization
   - Kubernetes for orchestration
   - CI/CD pipeline
   - Health checks and graceful shutdown
   - Environment-based configuration (dev/staging/prod)

5. **Error Handling**:
   - Retry logic with exponential backoff
   - Circuit breakers for external dependencies
   - Proper error responses and logging

---

## Testing Examples

### Part 1: Concurrent Rate Limit Test

```bash
# Test concurrent requests for the same user
# (Requires 'parallel' tool: apt-get install parallel)

# Generate 10 concurrent requests
seq 10 | parallel -j 10 curl -X POST http://localhost:8080/request \
  -H "Content-Type: application/json" \
  -d '{"user_id":"testuser","payload":{"request":{}}}'

# Check stats - should show 5 accepted, 5 rejected
curl http://localhost:8080/stats | jq '.stats.testuser'
```

### Part 2: Pagination Test

```bash
# Create multiple products
for i in {1..50}; do
  curl -X POST http://localhost:8080/products \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"Product $i\",\"sku\":\"SKU-$i\"}" \
    -s > /dev/null
done

# Test pagination
curl "http://localhost:8080/products?limit=10&offset=0" | jq '.products | length'  # Should return 10
curl "http://localhost:8080/products?limit=10&offset=10" | jq '.products | length' # Should return 10
curl "http://localhost:8080/products?limit=10&offset=40" | jq '.products | length' # Should return 10
curl "http://localhost:8080/products?limit=10&offset=50" | jq '.products | length' # Should return 0
```

### Load Testing

```bash
# Using Apache Bench (ab)
# Part 1: Rate limiter stress test
ab -n 1000 -c 50 -p request.json -T application/json http://localhost:8080/request

# Part 2: Product listing performance
ab -n 1000 -c 50 http://localhost:8080/products?limit=20

# request.json content:
# {"user_id":"loadtest","payload":{"test":true}}
```

---

## Project Structure

```
source-asia-backend/
├── main.go           # HTTP server and routing
├── models.go         # Data structures and types
├── ratelimiter.go    # Part 1 implementation
├── products.go       # Part 2 implementation
├── go.mod            # Go module definition
└── README.md         # This file
```

---

## Submission Notes

All requirements have been implemented:

**Part 1**:
-  POST /request with rate limiting (5 req/min per user)
-  GET /stats with per-user statistics
-  Concurrent-safe implementation
-  201/429/400 response codes
-  Fixed 1-minute window with sliding cleanup

**Part 2**:
-  POST /products with validation
-  GET /products with pagination (optimized list view)
-  GET /products/{id} with full details
-  POST /products/{id}/media
-  URL validation (http/https, max length, max count)
-  SKU uniqueness enforcement (409 Conflict)
-  List endpoint returns counts only (fast with many media URLs)

**General**:
-  Go implementation with standard library
-  In-memory storage
-  Concurrent-safe with mutexes
-  Comprehensive README with examples
-  Production limitations documented
-  Runnable HTTP service

**Build & Test Time**: ~2 minutes (run server, execute curl examples)

---

## Contact

For questions or clarifications about this implementation, please reach out via the assignment email thread.
