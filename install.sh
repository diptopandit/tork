#!/bin/sh
# tork installer — downloads the latest release and installs to /usr/local/bin.
# Usage: curl -sSfL https://raw.githubusercontent.com/diptopandit/tork/main/install.sh | sh

set -e

REPO="diptopandit/tork"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS
OS="$(uname -s)"
case "$OS" in
  Darwin)  OS="darwin" ;;
  Linux)   OS="linux" ;;
  MINGW*|MSYS*|CYGWIN*) OS="windows" ;;
  *) echo "Unsupported OS: $OS" >&2; exit 1 ;;
esac

# Detect architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)   ARCH="amd64" ;;
  arm64|aarch64)   ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Fetch latest release tag
echo "Fetching latest release..."
TAG=$(curl -sSf "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$TAG" ]; then
  echo "Error: could not determine latest release." >&2
  exit 1
fi
echo "Latest release: $TAG"

# Build download URL
if [ "$OS" = "windows" ]; then
  ARCHIVE="tork_${TAG}_${OS}_${ARCH}.zip"
else
  ARCHIVE="tork_${TAG}_${OS}_${ARCH}.tar.gz"
fi
URL="https://github.com/${REPO}/releases/download/${TAG}/${ARCHIVE}"

# Download and extract
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

echo "Downloading ${ARCHIVE}..."
curl -sSfL "$URL" -o "$TMPDIR/$ARCHIVE"

echo "Extracting..."
if [ "$OS" = "windows" ]; then
  unzip -qo "$TMPDIR/$ARCHIVE" -d "$TMPDIR"
else
  tar -xzf "$TMPDIR/$ARCHIVE" -C "$TMPDIR"
fi

# Find the extracted directory
EXTRACTED="$TMPDIR/tork_${TAG}_${OS}_${ARCH}"

# Install
echo "Installing to ${INSTALL_DIR}..."
if [ -w "$INSTALL_DIR" ]; then
  cp "$EXTRACTED/tork" "$INSTALL_DIR/tork"
  cp "$EXTRACTED/tork-cli" "$INSTALL_DIR/tork-cli"
  chmod +x "$INSTALL_DIR/tork" "$INSTALL_DIR/tork-cli"
else
  sudo cp "$EXTRACTED/tork" "$INSTALL_DIR/tork"
  sudo cp "$EXTRACTED/tork-cli" "$INSTALL_DIR/tork-cli"
  sudo chmod +x "$INSTALL_DIR/tork" "$INSTALL_DIR/tork-cli"
fi

echo ""
echo "tork ${TAG} installed successfully!"
echo "  tork      → ${INSTALL_DIR}/tork"
echo "  tork-cli  → ${INSTALL_DIR}/tork-cli"
echo ""
echo "Run 'tork --version' to verify."
