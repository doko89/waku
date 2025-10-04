# WAKU Installation Scripts

This directory contains automated installation, update, and uninstallation scripts for WAKU WhatsApp API.

## 📋 Available Scripts

### 1. `setup.sh` - Install WAKU

Automatically installs WAKU from the latest GitHub release.

**What it does:**
- ✅ Detects OS and architecture
- ✅ Downloads appropriate binary from latest release
- ✅ Installs system dependencies
- ✅ Creates service user and group
- ✅ Creates required directories
- ✅ Generates `.env` configuration file
- ✅ Sets up systemd service
- ✅ Configures permissions
- ✅ Enables and starts service

**Usage:**

```bash
# One-line installation
curl -fsSL https://raw.githubusercontent.com/doko89/waku/main/scripts/setup.sh | sudo bash

# Or download and run
wget https://raw.githubusercontent.com/doko89/waku/main/scripts/setup.sh
chmod +x setup.sh
sudo ./setup.sh
```

**Supported Platforms:**
- Linux: amd64, arm64, armv7
- macOS: amd64, arm64

**Installation Locations:**
- Binary: `/opt/waku/waku`
- Config: `/opt/waku/.env`
- Sessions: `/opt/waku/sessions/`
- Temp files: `/opt/waku/temp/`
- Logs: `/opt/waku/logs/`
- Service: `/etc/systemd/system/waku.service`

---

### 2. `update.sh` - Update WAKU

Updates WAKU to the latest version from GitHub releases.

**What it does:**
- ✅ Checks current version
- ✅ Fetches latest version
- ✅ Downloads new binary
- ✅ Backs up current binary
- ✅ Stops service
- ✅ Installs new binary
- ✅ Starts service
- ✅ Verifies update
- ✅ Auto-rollback on failure

**Usage:**

```bash
# One-line update
curl -fsSL https://raw.githubusercontent.com/doko89/waku/main/scripts/update.sh | sudo bash

# Or download and run
wget https://raw.githubusercontent.com/doko89/waku/main/scripts/update.sh
chmod +x update.sh
sudo ./update.sh
```

**Features:**
- Zero-downtime update (service restart only)
- Automatic backup before update
- Rollback on failure
- Keeps last 3 backups

---

### 3. `uninstall.sh` - Uninstall WAKU

Completely removes WAKU from the system.

**What it does:**
- ✅ Stops and disables service
- ✅ Removes systemd service
- ✅ Backs up session data and logs
- ✅ Removes installation directory
- ✅ Removes service user and group

**Usage:**

```bash
# One-line uninstallation
curl -fsSL https://raw.githubusercontent.com/doko89/waku/main/scripts/uninstall.sh | sudo bash

# Or download and run
wget https://raw.githubusercontent.com/doko89/waku/main/scripts/uninstall.sh
chmod +x uninstall.sh
sudo ./uninstall.sh
```

**Data Backup:**
Session data and logs are automatically backed up to:
```
/opt/waku.backup.YYYYMMDD_HHMMSS/
```

---

## 🚀 Quick Start

### Fresh Installation

```bash
# Install WAKU
curl -fsSL https://raw.githubusercontent.com/doko89/waku/main/scripts/setup.sh | sudo bash

# Configure API token
sudo nano /opt/waku/.env

# Restart service
sudo systemctl restart waku

# Check status
sudo systemctl status waku

# Test API
curl http://localhost:8080/qr/test-device
```

### Update Existing Installation

```bash
# Update to latest version
curl -fsSL https://raw.githubusercontent.com/doko89/waku/main/scripts/update.sh | sudo bash

# Check new version
sudo systemctl status waku
```

### Uninstall

```bash
# Remove WAKU (with data backup)
curl -fsSL https://raw.githubusercontent.com/doko89/waku/main/scripts/uninstall.sh | sudo bash
```

---

## 📝 Post-Installation Configuration

After installation, you need to configure WAKU:

### 1. Edit Configuration

```bash
sudo nano /opt/waku/.env
```

**Important settings:**

```env
# Change this to a secure token!
API_TOKEN=your-secure-token-here

# Server port (default: 8080)
PORT=8080

# Webhook configuration (optional)
WEBHOOK_URL=https://your-webhook-url.com/webhook
WEBHOOK_ENABLED=true
WEBHOOK_RETRY=3
```

### 2. Restart Service

```bash
sudo systemctl restart waku
```

### 3. Verify Service

```bash
# Check service status
sudo systemctl status waku

# View logs
sudo journalctl -u waku -f

# Or view log files
sudo tail -f /opt/waku/logs/waku.log
```

---

## 🔧 Service Management

### Start/Stop/Restart

```bash
# Start service
sudo systemctl start waku

# Stop service
sudo systemctl stop waku

# Restart service
sudo systemctl restart waku

# Check status
sudo systemctl status waku
```

### Enable/Disable Auto-start

```bash
# Enable auto-start on boot
sudo systemctl enable waku

# Disable auto-start
sudo systemctl disable waku
```

### View Logs

```bash
# Follow live logs
sudo journalctl -u waku -f

# View last 100 lines
sudo journalctl -u waku -n 100

# View logs since today
sudo journalctl -u waku --since today

# View log files directly
sudo tail -f /opt/waku/logs/waku.log
sudo tail -f /opt/waku/logs/waku-error.log
```

---

## 🔐 Security Recommendations

### 1. Change API Token

```bash
# Generate secure token
openssl rand -hex 32

# Update .env file
sudo nano /opt/waku/.env
# Set: API_TOKEN=<generated-token>

# Restart service
sudo systemctl restart waku
```

### 2. Configure Firewall

```bash
# Allow only from specific IP
sudo ufw allow from 192.168.1.0/24 to any port 8080

# Or allow from anywhere (not recommended for production)
sudo ufw allow 8080
```

### 3. Use Reverse Proxy

For production, use Nginx or Caddy as reverse proxy with HTTPS.

See [DEPLOYMENT.md](../DEPLOYMENT.md) for detailed instructions.

---

## 🐛 Troubleshooting

### Service Won't Start

```bash
# Check service status
sudo systemctl status waku

# View detailed logs
sudo journalctl -u waku -n 50 --no-pager

# Check if port is already in use
sudo lsof -i :8080

# Verify binary permissions
ls -la /opt/waku/waku

# Verify .env file exists
ls -la /opt/waku/.env
```

### Permission Errors

```bash
# Fix ownership
sudo chown -R waku:waku /opt/waku

# Fix permissions
sudo chmod 755 /opt/waku
sudo chmod 700 /opt/waku/sessions
sudo chmod 700 /opt/waku/temp
sudo chmod 755 /opt/waku/logs
sudo chmod 600 /opt/waku/.env
```

### Update Failed

```bash
# Check if backup exists
ls -la /opt/waku/waku.backup.*

# Manually restore backup
sudo cp /opt/waku/waku.backup.YYYYMMDD_HHMMSS /opt/waku/waku
sudo systemctl restart waku
```

---

## 📚 Additional Resources

- [Main README](../README.md) - Full documentation
- [API Documentation](../README.md#-api-documentation) - API endpoints
- [Deployment Guide](../DEPLOYMENT.md) - Production deployment
- [GitHub Repository](https://github.com/doko89/waku) - Source code

---

## 🆘 Support

If you encounter issues:

1. Check service logs: `sudo journalctl -u waku -f`
2. Verify configuration: `sudo cat /opt/waku/.env`
3. Check GitHub Issues: https://github.com/doko89/waku/issues
4. Open a new issue with logs and error messages

---

## 📄 License

These scripts are part of the WAKU project and follow the same license.

