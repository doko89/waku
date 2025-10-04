#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="doko89/waku"
APP_NAME="waku"
INSTALL_DIR="/opt/waku"
SYSTEMD_SERVICE="/etc/systemd/system/waku.service"

# Print colored message
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if running as root
check_root() {
    if [ "$EUID" -ne 0 ]; then
        print_error "This script must be run as root (use sudo)"
        exit 1
    fi
}

# Check if WAKU is installed
check_installation() {
    if [ ! -f "$INSTALL_DIR/$APP_NAME" ]; then
        print_error "WAKU is not installed at $INSTALL_DIR"
        print_info "Run setup.sh to install WAKU first"
        exit 1
    fi
}

# Get current version
get_current_version() {
    print_info "Checking current version..."
    
    if [ -f "$INSTALL_DIR/VERSION" ]; then
        CURRENT_VERSION=$(cat "$INSTALL_DIR/VERSION")
        print_info "Current version: $CURRENT_VERSION"
    else
        CURRENT_VERSION="unknown"
        print_warning "Current version unknown"
    fi
}

# Detect OS and Architecture
detect_platform() {
    print_info "Detecting platform..."
    
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    
    case "$OS" in
        linux*)
            OS="linux"
            ;;
        darwin*)
            OS="darwin"
            ;;
        *)
            print_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac
    
    case "$ARCH" in
        x86_64|amd64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l|armv7)
            ARCH="armv7"
            ;;
        *)
            print_error "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
    
    PLATFORM="${OS}-${ARCH}"
    print_success "Detected platform: $PLATFORM"
}

# Get latest release version
get_latest_version() {
    print_info "Fetching latest release version..."
    
    LATEST_VERSION=$(curl -s "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    
    if [ -z "$LATEST_VERSION" ]; then
        print_error "Failed to fetch latest version"
        exit 1
    fi
    
    print_success "Latest version: $LATEST_VERSION"
}

# Check if update is needed
check_update_needed() {
    if [ "$CURRENT_VERSION" = "$LATEST_VERSION" ]; then
        print_success "WAKU is already up to date ($CURRENT_VERSION)"
        exit 0
    fi
    
    print_info "Update available: $CURRENT_VERSION → $LATEST_VERSION"
}

# Download new binary
download_binary() {
    print_info "Downloading WAKU binary..."
    
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_VERSION}/${APP_NAME}-${PLATFORM}.tar.gz"
    TEMP_DIR=$(mktemp -d)
    
    print_info "Download URL: $DOWNLOAD_URL"
    
    if ! curl -L -o "${TEMP_DIR}/${APP_NAME}.tar.gz" "$DOWNLOAD_URL"; then
        print_error "Failed to download binary"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    print_info "Extracting binary..."
    tar -xzf "${TEMP_DIR}/${APP_NAME}.tar.gz" -C "$TEMP_DIR"
    
    # Find the binary
    BINARY_FILE=$(find "$TEMP_DIR" -type f -name "${APP_NAME}*" ! -name "*.tar.gz" ! -name "*.sha256" | head -1)
    
    if [ -z "$BINARY_FILE" ]; then
        print_error "Binary not found in archive"
        rm -rf "$TEMP_DIR"
        exit 1
    fi
    
    print_success "Binary downloaded and extracted"
    echo "$TEMP_DIR"
}

# Backup current binary
backup_binary() {
    print_info "Backing up current binary..."
    
    BACKUP_FILE="$INSTALL_DIR/${APP_NAME}.backup.$(date +%Y%m%d_%H%M%S)"
    cp "$INSTALL_DIR/$APP_NAME" "$BACKUP_FILE"
    
    print_success "Current binary backed up to: $BACKUP_FILE"
}

# Stop service
stop_service() {
    print_info "Stopping service..."
    
    if systemctl is-active --quiet waku.service; then
        systemctl stop waku.service
        print_success "Service stopped"
    else
        print_info "Service is not running"
    fi
}

# Install new binary
install_binary() {
    local TEMP_DIR=$1
    local BINARY_FILE=$(find "$TEMP_DIR" -type f -name "${APP_NAME}*" ! -name "*.tar.gz" ! -name "*.sha256" | head -1)
    
    print_info "Installing new binary..."
    
    cp "$BINARY_FILE" "$INSTALL_DIR/$APP_NAME"
    chmod +x "$INSTALL_DIR/$APP_NAME"
    chown waku:waku "$INSTALL_DIR/$APP_NAME"
    
    # Save version
    echo "$LATEST_VERSION" > "$INSTALL_DIR/VERSION"
    
    rm -rf "$TEMP_DIR"
    
    print_success "New binary installed"
}

# Start service
start_service() {
    print_info "Starting service..."
    
    systemctl start waku.service
    sleep 2
    
    if systemctl is-active --quiet waku.service; then
        print_success "Service started successfully"
    else
        print_error "Service failed to start"
        print_info "Check logs: sudo journalctl -u waku -n 50"
        exit 1
    fi
}

# Verify update
verify_update() {
    print_info "Verifying update..."
    
    sleep 3
    
    if systemctl is-active --quiet waku.service; then
        print_success "Service is running"
    else
        print_error "Service is not running after update"
        print_warning "Rolling back to previous version..."
        
        # Restore backup
        BACKUP_FILE=$(ls -t "$INSTALL_DIR/${APP_NAME}.backup."* 2>/dev/null | head -1)
        if [ -n "$BACKUP_FILE" ]; then
            cp "$BACKUP_FILE" "$INSTALL_DIR/$APP_NAME"
            systemctl start waku.service
            print_info "Rolled back to previous version"
        fi
        
        exit 1
    fi
}

# Clean old backups
clean_backups() {
    print_info "Cleaning old backups..."
    
    # Keep only last 3 backups
    ls -t "$INSTALL_DIR/${APP_NAME}.backup."* 2>/dev/null | tail -n +4 | xargs -r rm -f
    
    print_success "Old backups cleaned"
}

# Print summary
print_summary() {
    echo ""
    echo "========================================="
    echo -e "${GREEN}WAKU Update Complete!${NC}"
    echo "========================================="
    echo ""
    echo "Updated: $CURRENT_VERSION → $LATEST_VERSION"
    echo ""
    echo "Service Status:"
    systemctl status waku.service --no-pager -l | head -10
    echo ""
    echo "Useful Commands:"
    echo "  Check status:  sudo systemctl status waku"
    echo "  View logs:     sudo journalctl -u waku -f"
    echo "  Restart:       sudo systemctl restart waku"
    echo ""
    echo "========================================="
}

# Main update flow
main() {
    echo ""
    echo "========================================="
    echo "  WAKU WhatsApp API - Update Script"
    echo "========================================="
    echo ""
    
    check_root
    check_installation
    get_current_version
    detect_platform
    get_latest_version
    check_update_needed
    
    backup_binary
    stop_service
    
    TEMP_DIR=$(download_binary)
    install_binary "$TEMP_DIR"
    
    start_service
    verify_update
    clean_backups
    
    print_summary
}

# Run main function
main

