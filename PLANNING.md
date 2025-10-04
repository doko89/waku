# WAKU - WhatsApp API Complete Specification

Buatkan REST API WhatsApp bernama **WAKU** menggunakan **whatsmeow** (Go library) dan **Gin** framework dengan spesifikasi lengkap berikut:

## Overview
WAKU adalah WhatsApp API yang mendukung multi-device, memungkinkan 1 API instance mengelola multiple WhatsApp accounts secara bersamaan.

## Fitur Utama
1. ✅ Send message (personal)
2. ✅ Send group message
3. ✅ Send media message (personal)
4. ✅ Send group media message
5. ✅ Get contact list
6. ✅ Get group list
7. ✅ Session management (create, logout, delete)
8. ✅ QR Code pairing
9. ✅ Multi-device support
10. ✅ Webhook untuk incoming messages
11. ✅ API Token authentication

## Technical Stack
- **Language:** Go 1.21+
- **Framework:** Gin
- **WhatsApp Library:** Whatsmeow
- **Session Storage:** File-based (no database)
- **Config:** .env file
- **Media Storage:** Temporary upload → auto delete after send

## Project Structure
```
waku/
├── main.go
├── .env
├── .env.example
├── README.md
├── sessions/              # Session storage per device
│   ├── device001/
│   ├── device002/
│   └── ...
├── temp/                  # Temporary media uploads
├── handlers/              # HTTP handlers
│   ├── session.go
│   ├── message.go
│   ├── media.go
│   └── info.go
├── services/              # WhatsApp service logic
│   ├── whatsapp.go
│   └── webhook.go
├── middleware/            # Auth middleware
│   └── auth.go
└── utils/                 # Helper functions
    ├── response.go
    └── file.go
```

## Session Management

### Storage Strategy
- **File-based storage** - Tidak menggunakan database
- Session disimpan di `sessions/{device_id}/`
- Whatsmeow otomatis generate session files (protobuf/JSON format)
- Auto-load semua sessions saat server restart
- Multi-device: setiap device memiliki folder terpisah

### Session Lifecycle
1. **Create** → Initialize session folder & whatsmeow client
2. **QR Scan** → Pair device dengan WhatsApp
3. **Connected** → Session aktif dan ready untuk send/receive
4. **Logout** → Disconnect tapi session file tetap ada (bisa reconnect)
5. **Delete** → Hapus total session & files (harus scan QR ulang)

## API Endpoints

### Session Management

#### 1. Create Session
```http
POST /session/create
Authorization: Bearer {API_TOKEN}
Content-Type: application/json

Request Body:
{
  "device_id": "device001"
}

Response Success (200):
{
  "success": true,
  "message": "Session created successfully",
  "data": {
    "device_id": "device001",
    "qr_url": "/qr/device001",
    "status": "waiting_for_qr_scan"
  }
}

Response Error - Already Exists (400):
{
  "success": false,
  "message": "Session already exists for this device_id",
  "data": {
    "device_id": "device001",
    "status": "connected"
  }
}
```

#### 2. Get QR Code (PUBLIC - No Auth Required)
```http
GET /qr/:device_id

Response untuk Browser (Accept: text/html):
- HTML page dengan QR code image embedded
- Auto-refresh jika QR expired
- Show "Connected" message setelah berhasil scan

Response untuk API Client (Accept: application/json) (200):
{
  "success": true,
  "message": "QR code generated",
  "data": {
    "device_id": "device001",
    "qr_code": "2@abc123xyz...", // QR code string
    "expires_in": 60 // seconds
  }
}

Response Error - Session Not Found (404):
{
  "success": false,
  "message": "Session not found. Please create session first via POST /session/create",
  "data": null
}
```

#### 3. Logout Session
```http
POST /logout/:device_id
Authorization: Bearer {API_TOKEN}

Response (200):
{
  "success": true,
  "message": "Session logged out successfully",
  "data": {
    "device_id": "device001",
    "status": "disconnected"
  }
}

Note: Session files tetap ada, bisa reconnect tanpa scan QR ulang
```

#### 4. Delete Session
```http
DELETE /session/:device_id
Authorization: Bearer {API_TOKEN}

Response (200):
{
  "success": true,
  "message": "Session deleted successfully",
  "data": {
    "device_id": "device001",
    "deleted_at": "2025-10-04T10:30:00Z"
  }
}

Note: Hapus total session & files, harus scan QR ulang untuk reconnect
```

#### 5. Get Session Status
```http
GET /session/:device_id/status
Authorization: Bearer {API_TOKEN}

Response (200):
{
  "success": true,
  "message": "Session status retrieved",
  "data": {
    "device_id": "device001",
    "status": "connected", // waiting_for_qr_scan | connected | disconnected | not_found
    "phone": "628123456789",
    "connected_at": "2025-10-04T09:15:00Z"
  }
}
```

#### 6. List All Sessions
```http
GET /sessions
Authorization: Bearer {API_TOKEN}

Response (200):
{
  "success": true,
  "message": "Sessions retrieved",
  "data": {
    "total": 3,
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
      },
      {
        "device_id": "device003",
        "status": "waiting_for_qr_scan",
        "phone": null
      }
    ]
  }
}
```

### Messaging

#### 7. Send Personal Message
```http
POST /send
Authorization: Bearer {API_TOKEN}
Content-Type: application/json

Request Body:
{
  "device_id": "device001",
  "phone": "628123456789", // Format: country_code + number (no + or -)
  "message": "Hello from WAKU!"
}

Response (200):
{
  "success": true,
  "message": "Message sent successfully",
  "data": {
    "message_id": "3EB0XXXXX",
    "timestamp": 1696411200
  }
}
```

#### 8. Send Group Message
```http
POST /send-group
Authorization: Bearer {API_TOKEN}
Content-Type: application/json

Request Body:
{
  "device_id": "device001",
  "group_jid": "120363XXXXX@g.us", // Group JID from get groups
  "message": "Hello group!"
}

Response (200):
{
  "success": true,
  "message": "Group message sent successfully",
  "data": {
    "message_id": "3EB0XXXXX",
    "timestamp": 1696411200
  }
}
```

#### 9. Send Personal Media
```http
POST /send-media
Authorization: Bearer {API_TOKEN}
Content-Type: multipart/form-data

Form Data:
- device_id: "device001"
- phone: "628123456789"
- file: [binary file] // image, video, audio, or document
- caption: "Check this out!" (optional)

Response (200):
{
  "success": true,
  "message": "Media sent successfully",
  "data": {
    "message_id": "3EB0XXXXX",
    "media_type": "image",
    "file_size": 245678
  }
}

Supported Media Types:
- Image: jpg, jpeg, png, gif (max 16MB)
- Video: mp4, avi, mkv (max 64MB)
- Audio: mp3, ogg, m4a (max 16MB)
- Document: pdf, doc, docx, xls, xlsx, zip, etc (max 100MB)

Note: File otomatis dihapus dari temp folder setelah berhasil dikirim
```

#### 10. Send Group Media
```http
POST /send-group-media
Authorization: Bearer {API_TOKEN}
Content-Type: multipart/form-data

Form Data:
- device_id: "device001"
- group_jid: "120363XXXXX@g.us"
- file: [binary file]
- caption: "Check this out!" (optional)

Response (200):
{
  "success": true,
  "message": "Group media sent successfully",
  "data": {
    "message_id": "3EB0XXXXX",
    "media_type": "image",
    "file_size": 245678
  }
}
```

### Information

#### 11. Get Contacts
```http
GET /contacts/:device_id
Authorization: Bearer {API_TOKEN}

Response (200):
{
  "success": true,
  "message": "Contacts retrieved",
  "data": {
    "total": 150,
    "contacts": [
      {
        "jid": "628123456789@s.whatsapp.net",
        "name": "John Doe",
        "notify": "John", // WhatsApp display name
        "is_business": false
      },
      {
        "jid": "628987654321@s.whatsapp.net",
        "name": "Jane Smith",
        "notify": "Jane",
        "is_business": true
      }
    ]
  }
}
```

#### 12. Get Groups
```http
GET /groups/:device_id
Authorization: Bearer {API_TOKEN}

Response (200):
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
      },
      {
        "jid": "120363YYYYY@g.us",
        "name": "Work Team",
        "participants": 42,
        "is_admin": false
      }
    ]
  }
}
```

## Webhook Configuration

### Purpose
Webhook mengirimkan notifikasi real-time ke server eksternal saat WAKU menerima pesan masuk.

### Use Cases
- Auto-reply bot
- Message logging & analytics
- Integration dengan sistem lain (CRM, Telegram, Discord)
- Real-time notifications

### Webhook Flow
```
WhatsApp → WAKU API (receive message)
         ↓
WAKU sends POST to WEBHOOK_URL
         ↓
Your Server (process, auto-reply, save to DB, etc)
```

### Webhook Payload
```http
POST {WEBHOOK_URL}
Content-Type: application/json

{
  "device_id": "device001",
  "message_id": "3EB0XXXXX",
  "from": "628123456789@s.whatsapp.net",
  "from_name": "John Doe",
  "message": "Hello, I need info about your product",
  "message_type": "text", // text | image | video | audio | document
  "timestamp": 1696411200,
  "is_group": false,
  "group_jid": null, // filled if is_group = true
  "group_name": null,
  "media_url": null, // filled if message_type is media
  "quoted_message": null // filled if replying to another message
}
```

### Webhook Response
Your webhook endpoint should respond with 200 OK. WAKU will retry up to 3 times if webhook fails.

## Authentication & Security

### API Token
- Stored in `.env` file
- Required in `Authorization: Bearer {token}` header for all protected endpoints
- QR endpoint is public (no authentication) for easy access

### Middleware
```go
// Apply to all routes except /qr/*
func AuthMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        token := c.GetHeader("Authorization")
        expectedToken := "Bearer " + os.Getenv("API_TOKEN")
        
        if token != expectedToken {
            c.JSON(401, Response{
                Success: false,
                Message: "Unauthorized",
            })
            c.Abort()
            return
        }
        c.Next()
    }
}
```

## Environment Variables

### .env File
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

### .env.example
```env
API_TOKEN=change-this-to-secure-random-token
PORT=8080
SESSION_DIR=./sessions
TEMP_MEDIA_DIR=./temp
WEBHOOK_URL=https://example.com/webhook
WEBHOOK_ENABLED=true
WEBHOOK_RETRY=3
LOG_LEVEL=info
```

## Response Format Standard

### Success Response
```json
{
  "success": true,
  "message": "Descriptive success message",
  "data": {
    // Response data object
  }
}
```

### Error Response
```json
{
  "success": false,
  "message": "Descriptive error message",
  "data": null
}
```

### HTTP Status Codes
- `200` - Success
- `400` - Bad Request (invalid input)
- `401` - Unauthorized (invalid/missing API token)
- `404` - Not Found (session/resource not found)
- `500` - Internal Server Error

## Error Handling

### Common Errors

**Session Not Found:**
```json
{
  "success": false,
  "message": "Session not found for device_id: device001"
}
```

**Session Not Connected:**
```json
{
  "success": false,
  "message": "Session not connected. Please scan QR code first"
}
```

**Invalid Phone Number:**
```json
{
  "success": false,
  "message": "Invalid phone number format. Use: country_code + number (e.g., 628123456789)"
}
```

**File Too Large:**
```json
{
  "success": false,
  "message": "File size exceeds maximum limit for this media type"
}
```

**Webhook Failed:**
- Log error to console/file
- Retry up to 3 times with exponential backoff
- Continue processing (don't block message receive)

## Implementation Requirements

### Core Features
- ✅ Clean code architecture (handlers, services, middleware separation)
- ✅ Proper error handling with descriptive messages
- ✅ Logging for debugging (log incoming messages, API calls, errors)
- ✅ Concurrent handling untuk multiple devices
- ✅ Auto-reconnect session jika terputus
- ✅ QR code auto-refresh mechanism
- ✅ Media file validation (type, size)
- ✅ Auto cleanup temp media files
- ✅ Graceful shutdown (save all sessions before exit)

### Code Quality
- Use meaningful variable names
- Add comments for complex logic
- Handle edge cases (empty messages, invalid JID, etc)
- Validate all user inputs
- Use goroutines untuk webhook & message handling
- Proper mutex untuk concurrent session access

### Dependencies
```go
require (
    github.com/gin-gonic/gin v1.9.1
    github.com/joho/godotenv v1.5.1
    go.mau.fi/whatsmeow v0.0.0-latest
    google.golang.org/protobuf v1.31.0
)
```

## Deliverables

### 1. Complete Source Code
- All files following project structure
- Well-organized and commented
- Ready to run with `go run main.go`

### 2. README.md
Include:
- **Installation Steps**
  ```bash
  # Clone repository
  # Install dependencies
  go mod download
  
  # Setup environment
  cp .env.example .env
  # Edit .env with your configuration
  
  # Run application
  go run main.go
  ```

- **API Documentation**
  - All endpoints with examples
  - Request/response formats
  - Authentication guide

- **Usage Examples**
  ```bash
  # Create session
  curl -X POST http://localhost:8080/session/create \
    -H "Authorization: Bearer your-token" \
    -H "Content-Type: application/json" \
    -d '{"device_id": "device001"}'
  
  # Get QR code (open in browser)
  open http://localhost:8080/qr/device001
  
  # Send message
  curl -X POST http://localhost:8080/send \
    -H "Authorization: Bearer your-token" \
    -H "Content-Type: application/json" \
    -d '{
      "device_id": "device001",
      "phone": "628123456789",
      "message": "Hello!"
    }'
  ```

### 3. Additional Files
- `.env.example` - Template environment variables
- `.gitignore` - Exclude sessions/, temp/, .env
- `go.mod` & `go.sum` - Go dependencies

## Testing Checklist

Before delivery, ensure:
- ✅ Server starts without errors
- ✅ Can create session successfully
- ✅ QR code displays in browser (HTML)
- ✅ QR code returns JSON for API client
- ✅ Can scan QR and connect
- ✅ Can send personal message
- ✅ Can send group message
- ✅ Can send media (image, video, document)
- ✅ Can get contacts list
- ✅ Can get groups list
- ✅ Can logout session
- ✅ Can delete session
- ✅ Webhook sends payload on incoming message
- ✅ Multiple devices work simultaneously
- ✅ Session persists after server restart
- ✅ Proper error messages for all error cases

## Production Considerations

### Security
- Use strong API token (minimum 32 characters)
- Run behind reverse proxy (nginx) with HTTPS
- Implement rate limiting (optional)
- Add IP whitelist (optional)
- Regular session cleanup for inactive devices

### Performance
- Use goroutines for non-blocking operations
- Implement connection pooling
- Monitor memory usage for media uploads
- Set reasonable file size limits

### Monitoring
- Log all API requests
- Log message send/receive
- Monitor webhook failures
- Track session connections/disconnections

---

## Final Notes

**WAKU** is designed to be production-ready, scalable, and easy to maintain. The file-based session storage makes it simple to deploy without database dependencies, while still maintaining full WhatsApp functionality.

Build this API with best practices, clean code, and comprehensive error handling. Make it reliable, fast, and developer-friendly! 🚀