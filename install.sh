#!/bin/sh
set -e

# Sadr CLI Universal Installer
# Usage: curl -sSfL https://raw.githubusercontent.com/pedrohpereira74/sadr/main/install.sh | sh

OWNER="pedrohpereira74"
REPO="sadr"
BINARY="sadr"

# 1. Detect OS and Arch
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="x86_64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case $OS in
    linux) OS="Linux" ;;
    darwin) OS="Darwin" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# 2. Get latest version from GitHub API
VERSION=$(curl -s "https://api.github.com/repos/$OWNER/$REPO/releases/latest" | grep -Po '"tag_name": "v\K[^"]*')

if [ -z "$VERSION" ]; then
    echo "Could not find latest version. Check internet connection."
    exit 1
fi

echo "Installing sadr v$VERSION ($OS/$ARCH)..."

# 3. Construct URL (Matches GoReleaser template)
# Example: sadr_Darwin_x86_64.tar.gz
FILENAME="${BINARY}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$OWNER/$REPO/releases/download/v$VERSION/$FILENAME"

# 4. Download and Install
TMP_DIR=$(mktemp -d)
curl -sSfL "$URL" -o "$TMP_DIR/$FILENAME"

# Extract
tar -xzf "$TMP_DIR/$FILENAME" -C "$TMP_DIR"

# Install to /usr/local/bin
echo "Installing to /usr/local/bin (may require sudo)..."
if [ -w /usr/local/bin ]; then
    mv "$TMP_DIR/$BINARY" /usr/local/bin/
else
    sudo mv "$TMP_DIR/$BINARY" /usr/local/bin/
fi

# Cleanup
rm -rf "$TMP_DIR"

echo "sadr v$VERSION installed successfully!"
echo "Run 'sadr init' to get started."
