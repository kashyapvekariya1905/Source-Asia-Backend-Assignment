#!/bin/bash

# Comprehensive test script for the backend assignment
# Make sure the server is running before executing this script
# Usage: ./test_all.sh

BASE_URL="http://localhost:8080"

echo "=========================================="
echo "Source Asia Backend Assignment Tests"
echo "=========================================="
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

pass() {
  echo -e "${GREEN}✓${NC} $1"
}

fail() {
  echo -e "${RED}✗${NC} $1"
}

info() {
  echo -e "${YELLOW}ℹ${NC} $1"
}

# Check if server is running
echo "Checking server health..."
if curl -s "$BASE_URL/health" > /dev/null 2>&1; then
  pass "Server is running"
else
  fail "Server is not running. Start it with: go run ."
  exit 1
fi
echo ""

# ==========================================
# PART 1: Rate Limiter Tests
# ==========================================

echo "=========================================="
echo "PART 1: Rate Limiter Tests"
echo "=========================================="
echo ""

# Test 1: Accept valid requests
info "Test 1: Accepting valid requests (should accept first 5)..."
for i in {1..5}; do
  response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/request" \
    -H "Content-Type: application/json" \
    -d "{\"user_id\":\"alice\",\"payload\":{\"count\":$i}}")
  
  http_code=$(echo "$response" | tail -n 1)
  
  if [ "$http_code" -eq 201 ]; then
    pass "Request $i: Accepted (201)"
  else
    fail "Request $i: Expected 201, got $http_code"
  fi
done
echo ""

# Test 2: Rate limit enforcement
info "Test 2: Testing rate limit (6th request should be rejected)..."
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/request" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"alice","payload":{"count":6}}')

http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 429 ]; then
  pass "Rate limit enforced (429 Too Many Requests)"
else
  fail "Expected 429, got $http_code"
fi
echo ""

# Test 3: Invalid input validation
info "Test 3: Testing invalid input validation..."

# Empty user_id
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/request" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"","payload":{"test":1}}')
http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 400 ]; then
  pass "Empty user_id rejected (400)"
else
  fail "Empty user_id: Expected 400, got $http_code"
fi

# Missing user_id
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/request" \
  -H "Content-Type: application/json" \
  -d '{"payload":{"test":1}}')
http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 400 ]; then
  pass "Missing user_id rejected (400)"
else
  fail "Missing user_id: Expected 400, got $http_code"
fi

# Invalid JSON
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/request" \
  -H "Content-Type: application/json" \
  -d 'invalid json')
http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 400 ]; then
  pass "Invalid JSON rejected (400)"
else
  fail "Invalid JSON: Expected 400, got $http_code"
fi
echo ""

# Test 4: Statistics endpoint
info "Test 4: Checking statistics..."
stats=$(curl -s "$BASE_URL/stats")

alice_accepted=$(echo "$stats" | grep -o '"alice":{[^}]*"accepted_in_current_window":[0-9]*' | grep -o '[0-9]*$')
alice_rejected=$(echo "$stats" | grep -o '"alice":{[^}]*"rejected_cumulative":[0-9]*' | grep -o '[0-9]*$' | tail -n 1)

if [ "$alice_accepted" -eq 5 ]; then
  pass "Stats: alice has 5 accepted requests"
else
  fail "Stats: Expected 5 accepted, got $alice_accepted"
fi

if [ "$alice_rejected" -ge 1 ]; then
  pass "Stats: alice has $alice_rejected rejected request(s)"
else
  fail "Stats: Expected at least 1 rejected, got $alice_rejected"
fi
echo ""

# Test 5: Different users
info "Test 5: Testing different users (isolation)..."
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/request" \
  -H "Content-Type: application/json" \
  -d '{"user_id":"bob","payload":{"test":1}}')
http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 201 ]; then
  pass "Different user (bob) can make requests"
else
  fail "Different user: Expected 201, got $http_code"
fi
echo ""

# ==========================================
# PART 2: Product Catalog Tests
# ==========================================

echo "=========================================="
echo "PART 2: Product Catalog Tests"
echo "=========================================="
echo ""

# Test 6: Create product
info "Test 6: Creating a product..."
create_response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Keyboard",
    "sku": "KB-TEST-001",
    "image_urls": [
      "https://cdn.example.com/kb/img1.jpg",
      "https://cdn.example.com/kb/img2.jpg"
    ],
    "video_urls": [
      "https://cdn.example.com/kb/video1.mp4"
    ]
  }')

http_code=$(echo "$create_response" | tail -n 1)
product_body=$(echo "$create_response" | head -n -1)

if [ "$http_code" -eq 201 ]; then
  product_id=$(echo "$product_body" | grep -o '"id":"[^"]*"' | cut -d'"' -f4)
  pass "Product created (201) with ID: $product_id"
else
  fail "Create product: Expected 201, got $http_code"
fi
echo ""

# Test 7: Duplicate SKU
info "Test 7: Testing duplicate SKU rejection..."
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -d '{"name":"Duplicate","sku":"KB-TEST-001"}')
http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 409 ]; then
  pass "Duplicate SKU rejected (409 Conflict)"
else
  fail "Duplicate SKU: Expected 409, got $http_code"
fi
echo ""

# Test 8: Input validation
info "Test 8: Testing input validation..."

# Empty name
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -d '{"name":"","sku":"TEST-002"}')
http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 400 ]; then
  pass "Empty name rejected (400)"
else
  fail "Empty name: Expected 400, got $http_code"
fi

# Invalid URL
response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/products" \
  -H "Content-Type: application/json" \
  -d '{"name":"Test","sku":"TEST-003","image_urls":["not-a-url"]}')
http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 400 ]; then
  pass "Invalid URL rejected (400)"
else
  fail "Invalid URL: Expected 400, got $http_code"
fi
echo ""

# Test 9: List products
info "Test 9: Testing product listing..."
list_response=$(curl -s "$BASE_URL/products?limit=10&offset=0")

# Check that it doesn't contain full image_urls array
if echo "$list_response" | grep -q '"image_count"'; then
  pass "List returns image_count (not full array)"
else
  fail "List should return image_count"
fi

if echo "$list_response" | grep -q '"video_count"'; then
  pass "List returns video_count (not full array)"
else
  fail "List should return video_count"
fi

if echo "$list_response" | grep -q '"total"'; then
  pass "List returns pagination metadata"
else
  fail "List should return pagination metadata"
fi
echo ""

# Test 10: Get product detail
info "Test 10: Testing product detail endpoint..."
detail_response=$(curl -s "$BASE_URL/products/$product_id")

if echo "$detail_response" | grep -q '"image_urls":\['; then
  pass "Detail returns full image_urls array"
else
  fail "Detail should return full image_urls array"
fi

if echo "$detail_response" | grep -q '"video_urls":\['; then
  pass "Detail returns full video_urls array"
else
  fail "Detail should return full video_urls array"
fi
echo ""

# Test 11: Append media
info "Test 11: Testing media append..."
append_response=$(curl -s -w "\n%{http_code}" -X POST "$BASE_URL/products/$product_id/media" \
  -H "Content-Type: application/json" \
  -d '{
    "image_urls": ["https://cdn.example.com/kb/img3.jpg"],
    "video_urls": ["https://cdn.example.com/kb/video2.mp4"]
  }')

http_code=$(echo "$append_response" | tail -n 1)
append_body=$(echo "$append_response" | head -n -1)

if [ "$http_code" -eq 200 ]; then
  image_count=$(echo "$append_body" | grep -o '"image_urls":\[[^]]*\]' | grep -o 'https://' | wc -l)
  if [ "$image_count" -eq 3 ]; then
    pass "Media appended successfully (3 images total)"
  else
    fail "Expected 3 images, got $image_count"
  fi
else
  fail "Append media: Expected 200, got $http_code"
fi
echo ""

# Test 12: Non-existent product
info "Test 12: Testing non-existent product..."
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/products/99999")
http_code=$(echo "$response" | tail -n 1)

if [ "$http_code" -eq 404 ]; then
  pass "Non-existent product returns 404"
else
  fail "Non-existent product: Expected 404, got $http_code"
fi
echo ""

# Test 13: Pagination
info "Test 13: Creating multiple products for pagination test..."
for i in {1..25}; do
  curl -s -X POST "$BASE_URL/products" \
    -H "Content-Type: application/json" \
    -d "{\"name\":\"Product $i\",\"sku\":\"SKU-$(printf '%03d' $i)\"}" > /dev/null
done

page1=$(curl -s "$BASE_URL/products?limit=10&offset=0")
page1_count=$(echo "$page1" | grep -o '"id":"[^"]*"' | wc -l)

page2=$(curl -s "$BASE_URL/products?limit=10&offset=10")
page2_count=$(echo "$page2" | grep -o '"id":"[^"]*"' | wc -l)

if [ "$page1_count" -eq 10 ]; then
  pass "Page 1 has 10 products"
else
  fail "Page 1: Expected 10 products, got $page1_count"
fi

if [ "$page2_count" -eq 10 ]; then
  pass "Page 2 has 10 products"
else
  fail "Page 2: Expected 10 products, got $page2_count"
fi

if echo "$page1" | grep -q '"has_more":true'; then
  pass "Pagination indicates more results"
else
  fail "Pagination should indicate has_more:true"
fi
echo ""

# ==========================================
# Summary
# ==========================================

echo "=========================================="
echo "Test Summary"
echo "=========================================="
echo ""
echo "All critical tests completed!"
echo ""
echo "To test concurrency (requires GNU parallel):"
echo "  seq 10 | parallel -j 10 curl -X POST $BASE_URL/request -H 'Content-Type: application/json' -d '{\"user_id\":\"concurrent\",\"payload\":{}}'"
echo ""
echo "To create 1000 products for load testing:"
echo "  chmod +x seed_products.sh"
echo "  ./seed_products.sh"
echo ""
