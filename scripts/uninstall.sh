#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
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

# Confirm uninstallation
confirm_uninstall() {
    echo ""
    echo "========================================="
    echo "  WAKU WhatsApp API - Uninstall Script"
    echo "========================================="
    echo ""
    print_warning "This will remove:"
    echo "  - WAKU binary and installation directory"
    echo "  - Systemd service"
    echo "  - Service user and group"
    echo ""
    print_warning "Session data and logs will be preserved in: $INSTALL_DIR.backup"
    echo ""
    read -p "Are you sure you want to uninstall WAKU? (yes/no): " -r
    echo ""
    
    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        print_info "Uninstallation cancelled"
        exit 0
    fi
}

# Stop and disable service
stop_service() {
    print_info "Stopping and disabling service..."
    
    if systemctl is-active --quiet waku.service; then
        systemctl stop waku.service
        print_success "Service stopped"
    else
        print_info "Service is not running"
    fi
    
    if systemctl is-enabled --quiet waku.service 2>/dev/null; then
        systemctl disable waku.service
        print_success "Service disabled"
    else
        print_info "Service is not enabled"
    fi
}

# Remove systemd service
remove_service() {
    print_info "Removing systemd service..."
    
    if [ -f "$SYSTEMD_SERVICE" ]; then
        rm -f "$SYSTEMD_SERVICE"
        systemctl daemon-reload
        print_success "Systemd service removed"
    else
        print_info "Systemd service file not found"
    fi
}

# Backup data
backup_data() {
    print_info "Backing up session data and logs..."
    
    if [ -d "$INSTALL_DIR" ]; then
        BACKUP_DIR="${INSTALL_DIR}.backup.$(date +%Y%m%d_%H%M%S)"
        
        mkdir -p "$BACKUP_DIR"
        
        if [ -d "$INSTALL_DIR/sessions" ]; then
            cp -r "$INSTALL_DIR/sessions" "$BACKUP_DIR/"
        fi
        
        if [ -d "$INSTALL_DIR/logs" ]; then
            cp -r "$INSTALL_DIR/logs" "$BACKUP_DIR/"
        fi
        
        if [ -f "$INSTALL_DIR/.env" ]; then
            cp "$INSTALL_DIR/.env" "$BACKUP_DIR/"
        fi
        
        print_success "Data backed up to: $BACKUP_DIR"
    else
        print_info "Installation directory not found, skipping backup"
    fi
}

# Remove installation directory
remove_installation() {
    print_info "Removing installation directory..."
    
    if [ -d "$INSTALL_DIR" ]; then
        rm -rf "$INSTALL_DIR"
        print_success "Installation directory removed"
    else
        print_info "Installation directory not found"
    fi
}

# Remove user and group
remove_user() {
    print_info "Removing service user..."
    
    if id "$SERVICE_USER" &>/dev/null; then
        userdel "$SERVICE_USER" 2>/dev/null || true
        print_success "User $SERVICE_USER removed"
    else
        print_info "User $SERVICE_USER not found"
    fi
    
    if getent group "$SERVICE_GROUP" &>/dev/null; then
        groupdel "$SERVICE_GROUP" 2>/dev/null || true
        print_success "Group $SERVICE_GROUP removed"
    else
        print_info "Group $SERVICE_GROUP not found"
    fi
}

# Print summary
print_summary() {
    echo ""
    echo "========================================="
    echo -e "${GREEN}WAKU Uninstallation Complete!${NC}"
    echo "========================================="
    echo ""
    echo "Removed:"
    echo "  ✓ WAKU binary and installation"
    echo "  ✓ Systemd service"
    echo "  ✓ Service user and group"
    echo ""
    
    if [ -n "$BACKUP_DIR" ] && [ -d "$BACKUP_DIR" ]; then
        echo "Backup saved to:"
        echo "  $BACKUP_DIR"
        echo ""
        echo "To restore your data:"
        echo "  1. Reinstall WAKU"
        echo "  2. Copy sessions back:"
        echo "     sudo cp -r $BACKUP_DIR/sessions /opt/waku/"
        echo "  3. Copy .env back:"
        echo "     sudo cp $BACKUP_DIR/.env /opt/waku/"
        echo ""
    fi
    
    echo "To reinstall WAKU:"
    echo "  curl -fsSL https://raw.githubusercontent.com/doko89/waku/main/scripts/setup.sh | sudo bash"
    echo ""
    echo "========================================="
}

# Main uninstallation flow
main() {
    check_root
    confirm_uninstall
    
    stop_service
    remove_service
    backup_data
    remove_installation
    remove_user
    
    print_summary
}

# Run main function
main

