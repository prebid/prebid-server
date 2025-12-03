#!/bin/bash

# Download MaxMind GeoLite2-Country database
# Usage: ./download-maxmind.sh <OUTPUT_PATH>
# Requires: MAXMIND_LICENSE_KEY environment variable
# Optional: MAXMIND_ACCOUNT_ID (some MaxMind APIs may require it)

set -e

OUTPUT_PATH="${1:-/tmp/GeoLite2-Country.mmdb}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Check if file already exists
if [ -f "$OUTPUT_PATH" ]; then
    print_info "MaxMind database already exists at $OUTPUT_PATH"
    exit 0
fi

# Check for license key (required)
if [ -z "$MAXMIND_LICENSE_KEY" ]; then
    print_error "MAXMIND_LICENSE_KEY not set - cannot download database"
    exit 1
fi

# Account ID is optional for direct download API, but recommended
if [ -z "$MAXMIND_ACCOUNT_ID" ]; then
    print_warning "MAXMIND_ACCOUNT_ID not set (optional, but recommended)"
fi

print_info "Downloading MaxMind GeoLite2-Country database..."

# Create output directory if it doesn't exist
OUTPUT_DIR=$(dirname "$OUTPUT_PATH")
mkdir -p "$OUTPUT_DIR"

# Temporary directory for download
TMP_DIR=$(mktemp -d)
trap "rm -rf $TMP_DIR" EXIT

# Download URL - Account ID is optional for direct download API
# Some MaxMind services require both, but direct download API works with just license_key
if [ -n "$MAXMIND_ACCOUNT_ID" ]; then
    DOWNLOAD_URL="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&account_id=${MAXMIND_ACCOUNT_ID}&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
else
    DOWNLOAD_URL="https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&license_key=${MAXMIND_LICENSE_KEY}&suffix=tar.gz"
fi
TAR_FILE="${TMP_DIR}/GeoLite2-Country.tar.gz"

# Download the database
print_info "Fetching from MaxMind..."
print_info "URL: https://download.maxmind.com/app/geoip_download?edition_id=GeoLite2-Country&..."
HTTP_CODE=$(curl -s -w "%{http_code}" -f -L -o "$TAR_FILE" "$DOWNLOAD_URL" 2>&1) || true
if [ ! -f "$TAR_FILE" ] || [ ! -s "$TAR_FILE" ]; then
    print_error "Failed to download MaxMind database (HTTP: $HTTP_CODE)"
    print_error "Please verify your MAXMIND_ACCOUNT_ID and MAXMIND_LICENSE_KEY are correct"
    print_info "Get credentials at: https://www.maxmind.com/en/account"
    exit 1
fi
print_info "Download complete (HTTP: $HTTP_CODE), file size: $(du -h "$TAR_FILE" | cut -f1)"

# Extract the archive
print_info "Extracting database..."
if ! tar -xzf "$TAR_FILE" -C "$TMP_DIR"; then
    print_error "Failed to extract archive"
    exit 1
fi

# Find the .mmdb file (archive structure: GeoLite2-Country_YYYYMMDD/GeoLite2-Country.mmdb)
EXTRACTED_FILE=$(find "$TMP_DIR" -name "GeoLite2-Country.mmdb" -type f | head -1)
if [ -z "$EXTRACTED_FILE" ] || [ ! -f "$EXTRACTED_FILE" ]; then
    print_error "Database file not found in archive"
    exit 1
fi

# Move to final location
mv "$EXTRACTED_FILE" "$OUTPUT_PATH"

# Verify final file
if [ ! -f "$OUTPUT_PATH" ]; then
    print_error "Failed to create database file at $OUTPUT_PATH"
    exit 1
fi

# Set permissions (readable by all, writable by owner)
chmod 644 "$OUTPUT_PATH"

print_success "MaxMind database downloaded successfully to $OUTPUT_PATH"
print_info "File size: $(du -h "$OUTPUT_PATH" | cut -f1)"

exit 0

