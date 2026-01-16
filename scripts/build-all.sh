#!/bin/bash
# Build all services

set -e

echo "Building JobGipfel services..."

services=(
  "auth_service"
  "cv_generator"
  "autoapply_service"
  "job_search"
  "matching_service"
  "analytics_service"
  "scrapper"
)

for service in "${services[@]}"; do
  echo "Building $service..."
  cd "$service"
  go mod tidy
  go build -o "$service" ./cmd/server 2>/dev/null || go build -o "$service" ./cmd/scrapper 2>/dev/null
  cd ..
  echo "✓ $service built"
done

echo ""
echo "All services built successfully!"
