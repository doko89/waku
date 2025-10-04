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
SERVICE_USER="waku"
SERVICE_GROUP="waku"
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

# Download and extract binary
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

    print_success "Binary downloaded and extracted"
}

# Install dependencies
install_dependencies() {
    print_info "Installing dependencies..."
    
    if command -v apt-get &> /dev/null; then
        # Debian/Ubuntu
        apt-get update
        apt-get install -y curl ca-certificates
    elif command -v yum &> /dev/null; then
        # CentOS/RHEL
        yum install -y curl ca-certificates
    elif command -v apk &> /dev/null; then
        # Alpine
        apk add --no-cache curl ca-certificates
    else
        print_warning "Package manager not detected, skipping dependency installation"
    fi
    
    print_success "Dependencies installed"
}

# Create user and group
create_user() {
    print_info "Creating service user..."
    
    if id "$SERVICE_USER" &>/dev/null; then
        print_warning "User $SERVICE_USER already exists"
    else
        useradd -r -s /bin/false -d "$INSTALL_DIR" "$SERVICE_USER"
        print_success "User $SERVICE_USER created"
    fi
}

# Create directories
create_directories() {
    print_info "Creating directories..."
    
    mkdir -p "$INSTALL_DIR"
    mkdir -p "$INSTALL_DIR/sessions"
    mkdir -p "$INSTALL_DIR/temp"
    mkdir -p "$INSTALL_DIR/logs"
    
    print_success "Directories created"
}

# Install binary
install_binary() {
    print_info "Installing binary..."

    # Find the binary (it might have platform suffix)
    BINARY_FILE=$(find "$TEMP_DIR" -type f -name "${APP_NAME}*" ! -name "*.tar.gz" ! -name "*.sha256" | head -1)

    if [ -z "$BINARY_FILE" ]; then
        print_error "Binary not found in extracted files"
        ls -la "$TEMP_DIR"
        rm -rf "$TEMP_DIR"
        exit 1
    fi

    print_info "Found binary: $BINARY_FILE"

    cp "$BINARY_FILE" "$INSTALL_DIR/$APP_NAME"
    chmod +x "$INSTALL_DIR/$APP_NAME"

    rm -rf "$TEMP_DIR"

    print_success "Binary installed to $INSTALL_DIR/$APP_NAME"
}

# Create .env file
create_env_file() {
    print_info "Creating .env file..."
    
    if [ -f "$INSTALL_DIR/.env" ]; then
        print_warning ".env file already exists, creating .env.example instead"
        ENV_FILE="$INSTALL_DIR/.env.example"
    else
        ENV_FILE="$INSTALL_DIR/.env"
    fi
    
    cat > "$ENV_FILE" << 'EOF'
# WAKU WhatsApp API Configuration

# API Authentication Token
API_TOKEN=waku-secret-token-change-this-in-production

# Server Port
PORT=8080

# Session Storage Directory
SESSION_DIR=/opt/waku/sessions

# Temporary Media Directory
TEMP_MEDIA_DIR=/opt/waku/temp

# Webhook Configuration
WEBHOOK_URL=https://example.com/webhook
WEBHOOK_ENABLED=false
WEBHOOK_RETRY=3

# Logging
LOG_LEVEL=info
EOF
    
    chmod 600 "$ENV_FILE"
    
    if [ "$ENV_FILE" = "$INSTALL_DIR/.env" ]; then
        print_success ".env file created at $ENV_FILE"
        print_warning "IMPORTANT: Edit $ENV_FILE and change API_TOKEN!"
    else
        print_success ".env.example created at $ENV_FILE"
        print_warning "Copy .env.example to .env and configure it"
    fi
}

# Create systemd service
create_systemd_service() {
    print_info "Creating systemd service..."
    
    cat > "$SYSTEMD_SERVICE" << EOF
[Unit]
Description=WAKU WhatsApp API Service
After=network.target
Documentation=https://github.com/${REPO}

[Service]
Type=simple
User=${SERVICE_USER}
Group=${SERVICE_GROUP}
WorkingDirectory=${INSTALL_DIR}
ExecStart=${INSTALL_DIR}/${APP_NAME}
Restart=always
RestartSec=10
StandardOutput=append:${INSTALL_DIR}/logs/waku.log
StandardError=append:${INSTALL_DIR}/logs/waku-error.log

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=${INSTALL_DIR}/sessions ${INSTALL_DIR}/temp ${INSTALL_DIR}/logs

# Environment
EnvironmentFile=${INSTALL_DIR}/.env

[Install]
WantedBy=multi-user.target
EOF
    
    print_success "Systemd service created at $SYSTEMD_SERVICE"
}

# Set permissions
set_permissions() {
    print_info "Setting permissions..."
    
    chown -R ${SERVICE_USER}:${SERVICE_GROUP} "$INSTALL_DIR"
    chmod 755 "$INSTALL_DIR"
    chmod 700 "$INSTALL_DIR/sessions"
    chmod 700 "$INSTALL_DIR/temp"
    chmod 755 "$INSTALL_DIR/logs"
    
    print_success "Permissions set"
}

# Enable and start service
enable_service() {
    print_info "Enabling and starting service..."
    
    systemctl daemon-reload
    systemctl enable waku.service
    
    if [ -f "$INSTALL_DIR/.env" ]; then
        systemctl start waku.service
        print_success "Service started"
    else
        print_warning "Service enabled but not started (configure .env first)"
    fi
}

# Print summary
print_summary() {
    echo ""
    echo "========================================="
    echo -e "${GREEN}WAKU Installation Complete!${NC}"
    echo "========================================="
    echo ""
    echo "Installation Directory: $INSTALL_DIR"
    echo "Binary: $INSTALL_DIR/$APP_NAME"
    echo "Config: $INSTALL_DIR/.env"
    echo "Logs: $INSTALL_DIR/logs/"
    echo "Service: $SYSTEMD_SERVICE"
    echo ""
    echo "Next Steps:"
    echo "1. Edit configuration:"
    echo "   sudo nano $INSTALL_DIR/.env"
    echo ""
    echo "2. Start the service (if not started):"
    echo "   sudo systemctl start waku"
    echo ""
    echo "3. Check service status:"
    echo "   sudo systemctl status waku"
    echo ""
    echo "4. View logs:"
    echo "   sudo journalctl -u waku -f"
    echo "   # or"
    echo "   sudo tail -f $INSTALL_DIR/logs/waku.log"
    echo ""
    echo "5. Test API:"
    echo "   curl http://localhost:8080/qr/test-device"
    echo ""
    echo "========================================="
}

# Main installation flow
main() {
    echo ""
    echo "========================================="
    echo "  WAKU WhatsApp API - Setup Script"
    echo "========================================="
    echo ""

    check_root
    detect_platform
    get_latest_version
    install_dependencies

    # Create temp directory for download
    TEMP_DIR=$(mktemp -d)
    export TEMP_DIR

    download_binary

    create_user
    create_directories
    install_binary
    create_env_file
    create_systemd_service
    set_permissions
    enable_service

    print_summary
}

# Run main function
main

