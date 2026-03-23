#!/bin/bash

# Isolate Panel Development Script

set -e

cd "$(dirname "$0")/.."

echo "🚀 Starting Isolate Panel Development Environment..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo "❌ Error: Docker is not running"
    exit 1
fi

# Build and start containers
cd docker
docker-compose up --build

echo ""
echo "✅ Isolate Panel is running!"
echo "   Backend API: http://localhost:8080"
echo "   Frontend: http://localhost:5173"
echo ""
echo "Press Ctrl+C to stop"
