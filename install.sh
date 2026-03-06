#!/bin/sh
set -e

REPO="joaogustavo/tb"
BINARY="tb"
INSTALL_DIR="/usr/local/bin"

# --- Detect OS ---
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin|linux) ;;
    mingw*|cygwin*|msys*) OS="windows" ;;
    *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# --- Detect architecture ---
ARCH=$(uname -m)
case "$ARCH" in
    x86_64|amd64)   ARCH="amd64" ;;
    aarch64|arm64)  ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# --- Pick download tool ---
if command -v curl > /dev/null 2>&1; then
    fetch() { curl -fsSL "$1"; }
elif command -v wget > /dev/null 2>&1; then
    fetch() { wget -qO- "$1"; }
else
    echo "Error: curl or wget is required." >&2; exit 1
fi

# --- Resolve version ---
if [ -z "$VERSION" ]; then
    VERSION=$(fetch "https://api.github.com/repos/$REPO/releases/latest" \
        | grep '"tag_name"' | head -1 \
        | sed 's/.*"tag_name": *"\([^"]*\)".*/\1/')
fi

if [ -z "$VERSION" ]; then
    echo "Error: could not determine latest version. Set VERSION= to install a specific tag." >&2
    exit 1
fi

# --- Build download URL ---
EXT=""
[ "$OS" = "windows" ] && EXT=".exe"
URL="https://github.com/$REPO/releases/download/$VERSION/${BINARY}_${OS}_${ARCH}${EXT}"

# --- Download ---
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT

echo "Downloading $BINARY $VERSION ($OS/$ARCH)..."
if command -v curl > /dev/null 2>&1; then
    curl -fsSL "$URL" -o "$TMP"
else
    wget -qO "$TMP" "$URL"
fi
chmod +x "$TMP"

# --- Install ---
if [ -w "$INSTALL_DIR" ]; then
    mv "$TMP" "$INSTALL_DIR/$BINARY"
else
    echo "Installing to $INSTALL_DIR (sudo required)..."
    sudo mv "$TMP" "$INSTALL_DIR/$BINARY"
fi

echo "$BINARY $VERSION installed to $INSTALL_DIR/$BINARY"
echo "Run: $BINARY --version"
