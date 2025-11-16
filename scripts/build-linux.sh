#!/bin/bash

# Build script for SourceControl CLI on Linux/macOS

set -e

echo ""
echo "========================================"
echo "  Building SourceControl for Linux"
echo "========================================"
echo ""

# Set build directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
BUILD_DIR="$SCRIPT_DIR/../sourcecontrol"
OUTPUT_DIR="$HOME/.local/bin"
OUTPUT_FILE="$OUTPUT_DIR/srcc"

# Get version info
COMMIT_SHA=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%d_%H:%M:%S")
VERSION="0.1.0"

echo "[1/5] Cleaning previous builds..."
rm -f "$OUTPUT_FILE"

echo "[2/5] Creating output directory..."
mkdir -p "$OUTPUT_DIR"

echo "[3/5] Building executable..."
cd "$BUILD_DIR"
go build \
    -ldflags="-X main.Version=$VERSION -X main.BuildTime=$BUILD_TIME -X main.CommitSHA=$COMMIT_SHA" \
    -o "$OUTPUT_FILE" \
    ./cmd/sourcecontrol

echo "[4/5] Making executable..."
chmod +x "$OUTPUT_FILE"

echo "[5/5] Setting up environment..."
# Check if directory is already in PATH
if [[ ":$PATH:" != *":$OUTPUT_DIR:"* ]]; then
    echo ""
    echo "Adding to PATH..."

    # Detect shell and update appropriate config file
    if [ -n "$ZSH_VERSION" ]; then
        SHELL_CONFIG="$HOME/.zshrc"
    elif [ -n "$BASH_VERSION" ]; then
        if [ -f "$HOME/.bashrc" ]; then
            SHELL_CONFIG="$HOME/.bashrc"
        else
            SHELL_CONFIG="$HOME/.bash_profile"
        fi
    else
        SHELL_CONFIG="$HOME/.profile"
    fi

    # Add to PATH in shell config if not already there
    if ! grep -q "export PATH=\"\$HOME/.local/bin:\$PATH\"" "$SHELL_CONFIG" 2>/dev/null; then
        echo "" >> "$SHELL_CONFIG"
        echo "# Added by SourceControl build script" >> "$SHELL_CONFIG"
        echo "export PATH=\"\$HOME/.local/bin:\$PATH\"" >> "$SHELL_CONFIG"
        echo ""
        echo "⚠️  PATH updated in $SHELL_CONFIG"
        echo "   Run: source $SHELL_CONFIG"
        echo "   Or restart your terminal for changes to take effect."
    fi
else
    echo "PATH already configured."
fi

echo ""
echo "========================================"
echo "  ✅ Build completed successfully!"
echo "========================================"
echo ""
echo "Executable location: $OUTPUT_FILE"
echo ""
echo "You can now run:"
echo "  srcc --help"
echo ""
echo "If 'srcc' is not recognized, run:"
echo "  source $SHELL_CONFIG"
echo ""
