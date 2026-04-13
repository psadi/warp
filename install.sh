#!/usr/bin/env bash
set -euo pipefail

REPO="psadi/warp"
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64 | aarch64) ARCH="arm64" ;;
    *)
        echo "Unsupported arch: $ARCH"
        exit 1
        ;;
esac

VERSION="$(curl -s https://api.github.com/repos/$REPO/releases/latest | jq -r '.tag_name' | sed 's/^v//')"

echo "Installing warp v$VERSION for $OS/$ARCH"

BASE_URL="https://github.com/$REPO/releases/download/v$VERSION"

ASSET="warp-${VERSION}-${OS}-${ARCH}.tar.gz"

if [ "$OS" = "darwin" ]; then
    ASSET="warp-${VERSION}-${OS}-amd64.tar.gz"
fi

TMP="$(mktemp -d)"
cd "$TMP"

echo "Downloading $ASSET"
curl -fL "$BASE_URL/$ASSET" -o "$ASSET"

echo "Extracting..."
tar -xzf "$ASSET"

if [ "$OS" = "windows" ]; then
    BINARY="warp-${VERSION}-${OS}-${ARCH}.exe"
    DEST="$HOME/.local/bin/warp.exe"
else
    BINARY="warp-${VERSION}-${OS}-${ARCH}"
    DEST="$HOME/.local/bin/warp"
fi

mkdir -p "$HOME/.local/bin"
mv "$BINARY" "$DEST"
chmod +x "$DEST"

echo "Installed to $DEST"
echo "Done"
