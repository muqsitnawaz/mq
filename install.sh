#!/bin/bash
set -e

REPO="muqsitnawaz/mq"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) OS="darwin" ;;
  linux) OS="linux" ;;
  mingw*|msys*|cygwin*) OS="windows" ;;
  *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64|amd64) ARCH="amd64" ;;
  arm64|aarch64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

# Get latest version
VERSION=$(curl -sI "https://github.com/$REPO/releases/latest" | grep -i "location:" | sed 's/.*tag\///' | tr -d '\r\n')
if [ -z "$VERSION" ]; then
  echo "Failed to get latest version"
  exit 1
fi

echo "Installing mq $VERSION for $OS/$ARCH..."

# Download
EXT="tar.gz"
[ "$OS" = "windows" ] && EXT="zip"

URL="https://github.com/$REPO/releases/download/$VERSION/mq_${OS}_${ARCH}.${EXT}"
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

curl -sL "$URL" -o "$TMP_DIR/mq.$EXT"

# Extract
cd "$TMP_DIR"
if [ "$EXT" = "zip" ]; then
  unzip -q "mq.$EXT"
else
  tar -xzf "mq.$EXT"
fi

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv mq "$INSTALL_DIR/"
else
  echo "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv mq "$INSTALL_DIR/"
fi

echo "Installed mq to $INSTALL_DIR/mq"
echo "Run 'mq --help' to get started"
