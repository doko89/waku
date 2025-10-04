# WAKU - WhatsApp API

WAKU adalah WhatsApp REST API yang dibangun menggunakan **whatsmeow** (Go library) dan **Gin** framework. API ini mendukung multi-device, memungkinkan 1 instance API mengelola multiple WhatsApp accounts secara bersamaan.

## ✨ Fitur Utama

- ✅ Send message (personal & group)
- ✅ Send media message (image, video, audio, document)
- ✅ Get contact list
- ✅ Get group list
- ✅ Session management (create, logout, delete)
- ✅ QR Code pairing
- ✅ Multi-device support
- ✅ Webhook untuk incoming messages
- ✅ API Token authentication
- ✅ File-based session storage (no database required)

## 🚀 Installation

### Prerequisites

- Go 1.21 atau lebih baru
- Git
- Docker & Docker Compose (optional)

### Option 1: Run with Go

1. **Clone repository**
```bash
git clone <repository-url>
cd waku
```

2. **Install dependencies**
```bash
make install
# or
go mod download
```

3. **Setup environment**
```bash
make setup
# or manually:
cp .env.example .env
# Edit .env dengan konfigurasi Anda
```

4. **Run application**
```bash
make run
# or
go run main.go
```

Server akan berjalan di `http://localhost:8080`

### Option 2: Run with Docker

1. **Clone repository**
```bash
git clone <repository-url>
cd waku
```

2. **Setup environment**
```bash
cp .env.example .env
# Edit .env dengan konfigurasi Anda
```

3. **Run with Docker Compose**
```bash
docker-compose up -d
```

4. **View logs**
```bash
docker-compose logs -f
```

5. **Stop container**
```bash
docker-compose down
```

### Option 3: Pull from GitHub Container Registry

```bash
# Pull latest image
docker pull ghcr.io/YOUR_USERNAME/waku:latest

# Run container
docker run -d \
  -p 8080:8080 \
  -e API_TOKEN=your-secret-token \
  -v $(pwd)/sessions:/app/sessions \
  -v $(pwd)/temp:/app/temp \
  --name waku-api \
  ghcr.io/YOUR_USERNAME/waku:latest
```

## ⚙️ Configuration

Edit file `.env` untuk mengkonfigurasi aplikasi:

```env
# API Configuration
API_TOKEN=your-secret-api-token-change-this
PORT=8080

# Session Storage
SESSION_DIR=./sessions

# Media Storage
TEMP_MEDIA_DIR=./temp

# Webhook Configuration
WEBHOOK_URL=https://your-server.com/webhook
WEBHOOK_ENABLED=true
WEBHOOK_RETRY=3

# Logging
LOG_LEVEL=info  # debug | info | warn | error
```

### Important Notes:
- **API_TOKEN**: Gunakan token yang kuat (minimum 32 karakter) untuk production
- **WEBHOOK_URL**: URL endpoint yang akan menerima incoming messages
- **WEBHOOK_ENABLED**: Set `true` untuk mengaktifkan webhook

## 📚 API Documentation

### Authentication

Semua endpoint (kecuali `/qr/:device_id`) memerlukan authentication menggunakan Bearer token:

```bash
Authorization: Bearer your-api-token
```

### Endpoints

#### 1. Create Session

Membuat session baru untuk device WhatsApp.

```bash
POST /session/create
Authorization: Bearer {API_TOKEN}
Content-Type: application/json

{
  "device_id": "device001"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Session created successfully",
  "data": {
    "device_id": "device001",
    "qr_url": "/qr/device001",
    "status": "waiting_for_qr_scan"
  }
}
```

#### 2. Get QR Code

Mendapatkan QR code untuk pairing device. **Endpoint ini PUBLIC (tidak perlu authentication)**.

```bash
GET /qr/:device_id
```

**Browser (HTML):**
Buka di browser: `http://localhost:8080/qr/device001`

**API Client (JSON):**
```bash
curl -H "Accept: application/json" http://localhost:8080/qr/device001
```

**Response:**
```json
{
  "success": true,
  "message": "QR code generated",
  "data": {
    "device_id": "device001",
    "qr_code": "2@abc123xyz...",
    "expires_in": 60
  }
}
```

#### 3. Get Session Status

```bash
GET /session/:device_id/status
Authorization: Bearer {API_TOKEN}
```

**Response:**
```json
{
  "success": true,
  "message": "Session status retrieved",
  "data": {
    "device_id": "device001",
    "status": "connected",
    "phone": "628123456789",
    "connected_at": "2025-10-04T09:15:00Z"
  }
}
```

Status values: `waiting_for_qr_scan`, `connected`, `disconnected`, `not_found`

#### 4. List All Sessions

```bash
GET /sessions
Authorization: Bearer {API_TOKEN}
```

**Response:**
```json
{
  "success": true,
  "message": "Sessions retrieved",
  "data": {
    "total": 2,
    "sessions": [
      {
        "device_id": "device001",
        "status": "connected",
        "phone": "628123456789"
      },
      {
        "device_id": "device002",
        "status": "disconnected",
        "phone": "628987654321"
      }
    ]
  }
}
```

#### 5. Send Personal Message

```bash
POST /send
Authorization: Bearer {API_TOKEN}
Content-Type: application/json

{
  "device_id": "device001",
  "phone": "628123456789",
  "message": "Hello from WAKU!"
}
```

**Response:**
```json
{
  "success": true,
  "message": "Message sent successfully",
  "data": {
    "message_id": "3EB0XXXXX",
    "timestamp": 1696411200
  }
}
```

#### 6. Send Group Message

```bash
POST /send-group
Authorization: Bearer {API_TOKEN}
Content-Type: application/json

{
  "device_id": "device001",
  "group_jid": "120363XXXXX@g.us",
  "message": "Hello group!"
}
```

#### 7. Send Personal Media

```bash
POST /send-media
Authorization: Bearer {API_TOKEN}
Content-Type: multipart/form-data

Form Data:
- device_id: "device001"
- phone: "628123456789"
- file: [binary file]
- caption: "Check this out!" (optional)
```

**Response:**
```json
{
  "success": true,
  "message": "Media sent successfully",
  "data": {
    "message_id": "3EB0XXXXX",
    "media_type": "image",
    "file_size": 245678
  }
}
```

**Supported Media Types:**
- Image: jpg, jpeg, png, gif (max 16MB)
- Video: mp4, avi, mkv (max 64MB)
- Audio: mp3, ogg, m4a (max 16MB)
- Document: pdf, doc, docx, xls, xlsx, zip, etc (max 100MB)

#### 8. Send Group Media

```bash
POST /send-group-media
Authorization: Bearer {API_TOKEN}
Content-Type: multipart/form-data

Form Data:
- device_id: "device001"
- group_jid: "120363XXXXX@g.us"
- file: [binary file]
- caption: "Check this out!" (optional)
```

#### 9. Get Contacts

```bash
GET /contacts/:device_id
Authorization: Bearer {API_TOKEN}
```

**Response:**
```json
{
  "success": true,
  "message": "Contacts retrieved",
  "data": {
    "total": 150,
    "contacts": [
      {
        "jid": "628123456789@s.whatsapp.net",
        "name": "John Doe",
        "notify": "John",
        "is_business": false
      }
    ]
  }
}
```

#### 10. Get Groups

```bash
GET /groups/:device_id
Authorization: Bearer {API_TOKEN}
```

**Response:**
```json
{
  "success": true,
  "message": "Groups retrieved",
  "data": {
    "total": 25,
    "groups": [
      {
        "jid": "120363XXXXX@g.us",
        "name": "Family Group",
        "participants": 15,
        "is_admin": true
      }
    ]
  }
}
```

#### 11. Logout Session

```bash
POST /logout/:device_id
Authorization: Bearer {API_TOKEN}
```

Note: Session files tetap ada, bisa reconnect tanpa scan QR ulang.

#### 12. Delete Session

```bash
DELETE /session/:device_id
Authorization: Bearer {API_TOKEN}
```

Note: Hapus total session & files, harus scan QR ulang untuk reconnect.

## 🔔 Webhook

### Configuration

Set webhook di `.env`:
```env
WEBHOOK_URL=https://your-server.com/webhook
WEBHOOK_ENABLED=true
WEBHOOK_RETRY=3
```

### Webhook Payload

Saat ada pesan masuk, WAKU akan mengirim POST request ke `WEBHOOK_URL`:

```json
{
  "device_id": "device001",
  "message_id": "3EB0XXXXX",
  "from": "628123456789@s.whatsapp.net",
  "from_name": "John Doe",
  "message": "Hello, I need info about your product",
  "message_type": "text",
  "timestamp": 1696411200,
  "is_group": false,
  "group_jid": null,
  "group_name": null,
  "media_url": null,
  "quoted_message": null
}
```

### Webhook Response

Your webhook endpoint should respond with `200 OK`. WAKU will retry up to 3 times if webhook fails.

## 🧪 Testing

### Example: Complete Flow

```bash
# 1. Create session
curl -X POST http://localhost:8080/session/create \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{"device_id": "device001"}'

# 2. Get QR code (open in browser)
open http://localhost:8080/qr/device001

# 3. Check status
curl http://localhost:8080/session/device001/status \
  -H "Authorization: Bearer your-token"

# 4. Send message
curl -X POST http://localhost:8080/send \
  -H "Authorization: Bearer your-token" \
  -H "Content-Type: application/json" \
  -d '{
    "device_id": "device001",
    "phone": "628123456789",
    "message": "Hello!"
  }'

# 5. Send media
curl -X POST http://localhost:8080/send-media \
  -H "Authorization: Bearer your-token" \
  -F "device_id=device001" \
  -F "phone=628123456789" \
  -F "file=@/path/to/image.jpg" \
  -F "caption=Check this out!"
```

## 🏗️ Project Structure

```
waku/
├── main.go                          # Entry point
├── .env                             # Configuration
├── .env.example                     # Configuration template
├── README.md                        # Documentation
├── Dockerfile                       # Docker image definition
├── docker-compose.yml               # Docker Compose configuration
├── Makefile                         # Build automation
├── WAKU.postman_collection.json     # Postman API collection
├── go.mod                           # Go dependencies
├── go.sum                           # Go dependencies checksum
├── .github/
│   └── workflows/
│       ├── tag.yml                  # Auto-tag workflow
│       └── build.yml                # Build & release workflow
├── sessions/                        # Session storage per device
├── temp/                            # Temporary media uploads
├── handlers/                        # HTTP handlers
│   ├── session.go
│   ├── message.go
│   ├── media.go
│   └── info.go
├── services/                        # WhatsApp service logic
│   ├── whatsapp.go
│   └── webhook.go
├── middleware/                      # Auth middleware
│   └── auth.go
└── utils/                           # Helper functions
    ├── response.go
    └── file.go
```

## 🛠️ Development

### Using Makefile

```bash
# Show all available commands
make help

# Setup development environment
make setup

# Run in development mode
make dev

# Build binary
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run tests with coverage
make test-coverage

# Format code
make fmt

# Run linter
make lint

# Docker commands
make docker-build
make docker-run
make docker-stop
make docker-logs
make docker-clean
```

## 🚢 CI/CD

### Creating a Release

1. **Manual Tag Creation** (via GitHub Actions):
   - Go to Actions tab in GitHub
   - Select "Create Tag" workflow
   - Click "Run workflow"
   - Enter version (e.g., `1.0.0`)
   - Click "Run workflow"

2. **Automatic Build**:
   - When tag `v*.*.*` is pushed, build workflow automatically runs
   - Builds binaries for multiple platforms:
     - Linux (amd64, arm64, arm/v7)
     - macOS (amd64, arm64)
     - Windows (amd64)
   - Builds multi-arch Docker images
   - Creates GitHub Release with all artifacts

### Docker Images

Multi-architecture Docker images are automatically built and pushed to GitHub Container Registry:

```bash
# Pull specific version
docker pull ghcr.io/YOUR_USERNAME/waku:1.0.0

# Pull latest
docker pull ghcr.io/YOUR_USERNAME/waku:latest
```

Supported platforms:
- `linux/amd64`
- `linux/arm64`
- `linux/arm/v7`

## 📦 Postman Collection

Import `WAKU.postman_collection.json` ke Postman untuk testing API:

1. Open Postman
2. Click Import
3. Select `WAKU.postman_collection.json`
4. Update environment variables:
   - `BASE_URL`: Your API URL
   - `API_TOKEN`: Your API token
   - `DEVICE_ID`: Your device ID

## 🔒 Security

### Production Recommendations:

1. **Strong API Token**: Use minimum 32 characters random token
2. **HTTPS**: Run behind reverse proxy (nginx) with SSL/TLS
3. **Rate Limiting**: Implement rate limiting to prevent abuse
4. **IP Whitelist**: Restrict access to known IPs (optional)
5. **Regular Cleanup**: Clean up inactive sessions periodically

## 📝 License

MIT License

## 🤝 Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## 📧 Support

For issues and questions, please open an issue on GitHub.

---

**Built with ❤️ using Go, Gin, and Whatsmeow**

