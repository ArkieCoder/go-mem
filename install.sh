#!/bin/bash

set -euo pipefail

# Constants
REPO="ArkieCoder/go-mem"
DEFAULT_INSTALL_PATH="/usr/local/bin/go-mem"
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Logging functions
log_info() { echo -e "${GREEN}INFO:${NC} $1"; }
log_warn() { echo -e "${YELLOW}WARN:${NC} $1" >&2; }
log_error() { echo -e "${RED}ERROR:${NC} $1" >&2; }

# Check if required tools are available
check_tools() {
    local missing=()
    for tool in curl tar; do
        if ! command -v "$tool" >/dev/null 2>&1; then
            missing+=("$tool")
        fi
    done

    # Check for sha256sum or shasum
    if ! command -v sha256sum >/dev/null 2>&1 && ! command -v shasum >/dev/null 2>&1; then
        missing+=("sha256sum or shasum")
    fi

    if [ ${#missing[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing[*]}"
        log_error "Please install them and try again."
        exit 1
    fi

    # Check for jq (optional, improves JSON parsing)
    if command -v jq >/dev/null 2>&1; then
        USE_JQ=true
        log_info "jq found, using for JSON parsing"
    else
        USE_JQ=false
        log_warn "jq not found, falling back to grep for JSON parsing"
    fi
}

# Detect OS
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux) echo "linux" ;;
        darwin) echo "darwin" ;;
        *)
            log_error "Unsupported OS: $os"
            exit 2
            ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64) echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        i386|i686) echo "386" ;;
        *)
            log_error "Unsupported architecture: $arch"
            exit 2
            ;;
    esac
}

# Get release info from GitHub API
get_release_info() {
    local version=$1
    local api_url

    if [ "$version" = "latest" ]; then
        api_url="https://api.github.com/repos/$REPO/releases/latest"
    else
        api_url="https://api.github.com/repos/$REPO/releases/tags/$version"
    fi

    curl -s "$api_url"
}

# Extract asset URLs from release JSON
get_asset_urls() {
    local release_json=$1
    local os=$2
    local arch=$3
    local version

    if [ "$USE_JQ" = true ]; then
        version=$(echo "$release_json" | jq -r '.tag_name' | sed 's/^v//')
        local archive_pattern="go-mem_${version}_${os}_${arch}.tar.gz"
        local archive_url
        archive_url=$(echo "$release_json" | jq -r ".assets[] | select(.name == \"$archive_pattern\") | .browser_download_url")
        local checksum_pattern="go-mem_${version}_checksums.txt"
        local checksum_url
        checksum_url=$(echo "$release_json" | jq -r ".assets[] | select(.name == \"$checksum_pattern\") | .browser_download_url")
    else
        version=$(echo "$release_json" | grep -o '"tag_name": "[^"]*"' | cut -d'"' -f4 | sed 's/^v//')
        local archive_pattern="go-mem_${version}_${os}_${arch}\.tar\.gz"
        local archive_url
        archive_url=$(echo "$release_json" | grep "$archive_pattern" | grep '"browser_download_url"' | head -1 | cut -d'"' -f4)
        local checksum_pattern="go-mem_${version}_checksums\.txt"
        local checksum_url
        checksum_url=$(echo "$release_json" | grep "$checksum_pattern" | grep '"browser_download_url"' | head -1 | cut -d'"' -f4)
    fi

    if [ -z "$archive_url" ]; then
        log_error "No matching archive found for $os/$arch"
        exit 3
    fi

    if [ -z "$checksum_url" ]; then
        log_error "No checksum file found"
        exit 3
    fi

    echo "$archive_url $checksum_url"
}

# Download file
download_file() {
    local url=$1
    local dest=$2
    log_info "Downloading $url"
    if ! curl -L -o "$dest" "$url"; then
        log_error "Failed to download $url"
        exit 3
    fi
}

# Verify checksum
verify_checksum() {
    local file=$1
    local checksum_file=$2
    local filename
    filename=$(basename "$file")

    local expected_hash
    expected_hash=$(grep "$filename" "$checksum_file" | awk '{print $1}')

    if [ -z "$expected_hash" ]; then
        log_error "Checksum not found for $filename"
        exit 4
    fi

    local actual_hash
    if command -v sha256sum >/dev/null 2>&1; then
        actual_hash=$(sha256sum "$file" | awk '{print $1}')
    else
        actual_hash=$(shasum -a 256 "$file" | awk '{print $1}')
    fi

    if [ "$actual_hash" != "$expected_hash" ]; then
        log_error "Checksum mismatch for $filename"
        log_error "Expected: $expected_hash"
        log_error "Actual: $actual_hash"
        exit 4
    fi

    log_info "Checksum verified for $filename"
}

# Extract archive
extract_archive() {
    local archive=$1
    local dest_dir=$2
    log_info "Extracting $archive"
    tar -xzf "$archive" -C "$dest_dir"
}

# Install binary
install_binary() {
    local binary_path=$1
    local install_path=$2

    local install_dir
    install_dir=$(dirname "$install_path")

    # Check if install directory is writable
    if [ ! -w "$install_dir" ]; then
        log_warn "Install directory $install_dir is not writable. Using sudo."
        if ! sudo mv "$binary_path" "$install_path"; then
            log_error "Failed to install with sudo"
            exit 5
        fi
    else
        if ! mv "$binary_path" "$install_path"; then
            log_error "Failed to install"
            exit 5
        fi
    fi

    chmod +x "$install_path"
    log_info "Installed go-mem to $install_path"
}

# Main function
main() {
    local version=${1:-latest}
    local install_path=${2:-$DEFAULT_INSTALL_PATH}

    log_info "Installing go-mem version: $version"

    check_tools

    local os arch
    os=$(detect_os)
    arch=$(detect_arch)
    log_info "Detected platform: $os/$arch"

    local release_json
    release_json=$(get_release_info "$version")

    if [ -z "$release_json" ] || echo "$release_json" | grep -q '"message": "Not Found"'; then
        log_error "Release $version not found"
        exit 3
    fi

    local asset_urls
    asset_urls=$(get_asset_urls "$release_json" "$os" "$arch")
    local archive_url checksum_url
    read -r archive_url checksum_url <<< "$asset_urls"

    local archive_file checksum_file
    archive_file="$TEMP_DIR/$(basename "$archive_url")"
    checksum_file="$TEMP_DIR/$(basename "$checksum_url")"

    download_file "$archive_url" "$archive_file"
    download_file "$checksum_url" "$checksum_file"

    verify_checksum "$archive_file" "$checksum_file"

    extract_archive "$archive_file" "$TEMP_DIR"

    local binary_path="$TEMP_DIR/go-mem"
    if [ ! -f "$binary_path" ]; then
        log_error "Binary not found in archive"
        exit 5
    fi

    install_binary "$binary_path" "$install_path"

    log_info "Installation complete! Run 'go-mem --help' to get started."
}

main "$@"