#!/bin/bash

# Seed script to create 1000 products for performance testing
# Usage: ./seed_products.sh

echo "Creating 1000 products with 10 images each..."
echo "This will take a few minutes..."
echo ""

BASE_URL="http://localhost:8080"
TOTAL_PRODUCTS=1000

for i in $(seq 1 $TOTAL_PRODUCTS); do
  curl -X POST "$BASE_URL/products" \
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
  
  # Progress indicator
  if [ $((i % 100)) -eq 0 ]; then
    echo "Created $i products..."
  fi
done

echo ""
echo "Done! Created $TOTAL_PRODUCTS products."
echo ""
echo "Testing GET /products?limit=20 performance..."
time curl -s "$BASE_URL/products?limit=20" | grep -o '"products":\[' > /dev/null && echo "✓ List endpoint works"

echo ""
echo "Checking total products..."
curl -s "$BASE_URL/products?limit=1" | grep -o '"total":[0-9]*' | cut -d: -f2

echo ""
echo "Testing pagination at offset 500..."
curl -s "$BASE_URL/products?limit=10&offset=500" | grep -o '"id":"[0-9]*"' | head -n 1
