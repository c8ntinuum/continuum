#!/bin/bash

# JSON-RPC Compatibility Test Runner with Docker Image Optimization
# This script handles Docker image building with content-based caching

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"
JSONRPC_DIR="$PROJECT_ROOT/tests/jsonrpc"

echo "🔍 Checking Docker image requirements..."

# Check ctmd image and build if needed
if ! docker image inspect cosmos/evmd >/dev/null 2>&1; then
    echo "📦 Building cosmos/evmd image..."
    make -C "$PROJECT_ROOT" localnet-build-env
else
    echo "✓ cosmos/evmd image already exists, skipping build"
fi

# Check if simulator image already exists
if docker image inspect jsonrpc_simulator >/dev/null 2>&1; then
    echo "✓ Simulator image already exists"
else
    echo "📦 Will build simulator image..."
fi

# Initialize ctmd data directory
echo "🔧 Preparing ctmd data directory..."

# Clear existing directory to avoid key conflicts
if [ -d "$JSONRPC_DIR/.ctmd" ]; then
    echo "🧹 Removing existing .ctmd directory..."
    rm -rf "$JSONRPC_DIR/.ctmd"
fi

# Create fresh directory with correct permissions  
mkdir -p "$JSONRPC_DIR/.ctmd"
chmod 777 "$JSONRPC_DIR/.ctmd"

echo "🔧 ctmd will auto-initialize when container starts..."

# Run the compatibility tests
echo "🚀 Running JSON-RPC compatibility tests..."
cd "$JSONRPC_DIR" && docker compose up --build --abort-on-container-exit


echo "✅ JSON-RPC compatibility test completed!"
