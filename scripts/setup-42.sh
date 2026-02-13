#!/bin/bash
# Setup Go 1.25.6 on 42 machines (Ubuntu) without touching system Go
# Usage: bash scripts/setup-42.sh
#
# This script:
# - Installs Go 1.25.6 in ~/.poly-go-sdk/ (isolated, won't affect anything else)
# - Builds Poly using that Go
# - Does NOT modify your .zshrc, .bashrc, or system PATH
# - Does NOT remove or replace the system Go

set -e

GO_VERSION="1.25.6"
SDK_DIR="$HOME/.poly-go-sdk"
TARBALL="go${GO_VERSION}.linux-amd64.tar.gz"
URL="https://go.dev/dl/${TARBALL}"

echo "=== Poly Go Setup for 42 ==="
echo ""

# Check if already installed
if [ -x "$SDK_DIR/go/bin/go" ]; then
    INSTALLED=$("$SDK_DIR/go/bin/go" version 2>/dev/null | grep -oP 'go[\d.]+' || echo "unknown")
    echo "Go already installed in $SDK_DIR ($INSTALLED)"
    echo "To reinstall, delete $SDK_DIR and re-run this script."
else
    echo "Downloading Go $GO_VERSION..."
    cd /tmp
    curl -LO "$URL"

    echo "Installing to $SDK_DIR..."
    mkdir -p "$SDK_DIR"
    tar -C "$SDK_DIR" -xzf "$TARBALL"
    rm -f "$TARBALL"

    echo "Go $GO_VERSION installed in $SDK_DIR/go/"
fi

echo ""

# Build Poly
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

echo "Building Poly..."
cd "$PROJECT_DIR"
"$SDK_DIR/go/bin/go" build -o poly .

echo ""
echo "=== Done! ==="
echo "Binary: $PROJECT_DIR/poly"
echo ""
echo "To run:  ./poly"
echo ""
echo "Optional: add this to your shell to use this Go elsewhere:"
echo "  export PATH=$SDK_DIR/go/bin:\$PATH"
