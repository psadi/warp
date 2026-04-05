#!/bin/bash
set -e

REPO="psadi/warp"
FZF_REPO="junegunn/fzf"
INSTALL_DIR="${HOME}/.local/bin"
BINARY_NAME="warp"

detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        linux*) OS="linux" ;;
        darwin*) OS="darwin" ;;
        mingw*|msys*|cygwin*) OS="windows" ;;
        *) echo "Unsupported OS: $OS"; exit 1 ;;
    esac
    echo "$OS"
}

detect_arch() {
    ARCH=$(uname -m | tr '[:upper:]' '[:lower:]')
    case "$ARCH" in
        x86_64|amd64) ARCH="amd64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l) ARCH="armv7" ;;
        *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    echo "$ARCH"
}

get_latest_version() {
    LATEST=$(curl -s "https://api.github.com/repos/${1}/releases/latest" | grep -o '"tag_name":.*' | cut -d'"' -f4 | sed 's/v//')
    if [ -z "$LATEST" ]; then
        echo "Failed to get latest version" >&2
        exit 1
    fi
    echo "$LATEST"
}

install_fzf() {
    FZF_VERSION=$(get_latest_version "$FZF_REPO")
    OS=$(detect_os)
    ARCH=$(detect_arch)

    EXT=""
    if [ "$OS" = "windows" ]; then
        EXT=".exe"
    fi

    if [ "$OS" = "windows" ]; then
        FILENAME="fzf-${FZF_VERSION}-${OS}-${ARCH}.zip"
        URL="https://github.com/${FZF_REPO}/releases/download/v${FZF_VERSION}/${FILENAME}"
        TEMP_DIR=$(mktemp -d)
        curl -sL "$URL" -o "${TEMP_DIR}/fzf.zip"
        unzip -o "${TEMP_DIR}/fzf.zip" -d "$INSTALL_DIR"
        rm -rf "$TEMP_DIR"
    else
        FILENAME="fzf-${FZF_VERSION}-${OS}_${ARCH}.tar.gz"
        URL="https://github.com/${FZF_REPO}/releases/download/v${FZF_VERSION}/${FILENAME}"
        TEMP_DIR=$(mktemp -d)
        curl -sL "$URL" -o "${TEMP_DIR}/fzf.tar.gz"
        tar -xzf "${TEMP_DIR}/fzf.tar.gz" -C "$INSTALL_DIR"
        rm -rf "$TEMP_DIR"
    fi

    echo "✓ Installed fzf v${FZF_VERSION}"
}

get_warp_url() {
    OS=$1
    ARCH=$2
    URLS=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep "browser_download_url" | cut -d'"' -f4)
    for url in $URLS; do
        if [[ "$url" == *"${OS}"* ]] && [[ "$url" == *"${ARCH}"* ]]; then
            echo "$url"
            return 0
        fi
    done
    return 1
}

install() {
    OS=$(detect_os)
    ARCH=$(detect_arch)
    VERSION=$(get_latest_version "$REPO")

    echo "Installing warp v${VERSION} for ${OS}/${ARCH}..."

    DOWNLOAD_URL=$(get_warp_url "$OS" "$ARCH")
    if [ -z "$DOWNLOAD_URL" ]; then
        echo "No release found for ${OS}/${ARCH}" >&2
        exit 1
    fi

    FILENAME=$(basename "$DOWNLOAD_URL")
    EXT=""
    if [[ "$FILENAME" == *.exe ]]; then
        EXT=".exe"
    fi

    TEMP_DIR=$(mktemp -d)
    cd "$TEMP_DIR"

    echo "Downloading ${FILENAME}..."
    curl -sL "$DOWNLOAD_URL" -o "$FILENAME"

    if [ ! -f "$FILENAME" ]; then
        echo "Download failed." >&2
        exit 1
    fi

    chmod +x "$FILENAME"

    mkdir -p "$INSTALL_DIR"
    mv "$FILENAME" "${INSTALL_DIR}/${BINARY_NAME}${EXT}"

    cd ~
    rm -rf "$TEMP_DIR"

    echo ""
    echo "✓ Installed warp v${VERSION} to ${INSTALL_DIR}/"
    echo ""

    if command -v fzf &> /dev/null; then
        echo "✓ fzf is installed"
    else
        echo "fzf is required but not installed."
        read -p "Install fzf? [Y/n] " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Nn]$ ]]; then
            install_fzf
        else
            echo "Skipped. Install fzf manually to use warp."
        fi
    fi

    echo ""
    echo "Add ${INSTALL_DIR} to your PATH if needed, then run: warp"
}

uninstall() {
    echo "Uninstalling warp..."
    rm -f "${INSTALL_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}.exe" "${INSTALL_DIR}/fzf" "${INSTALL_DIR}/fzf.exe" 2>/dev/null
    echo "Done."
}

case "${1:-install}" in
    install|i) install ;;
    uninstall|uninstall|u) uninstall ;;
    *)
        echo "Usage: $0 [install|uninstall]"
        exit 1
        ;;
esac
