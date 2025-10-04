package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"
	"waku/services"
	"waku/utils"

	"github.com/gin-gonic/gin"
)

// CreateSessionRequest represents the request body for creating a session
type CreateSessionRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
}

// CreateSession creates a new WhatsApp session
func CreateSession(c *gin.Context) {
	var req CreateSessionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	waService := services.GetWhatsAppService()

	// Create session
	deviceClient, err := waService.CreateSession(req.DeviceID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Setup webhook event handler
	services.SetupEventHandler(req.DeviceID, deviceClient)

	utils.SuccessResponse(c, http.StatusOK, "Session created successfully", gin.H{
		"device_id": req.DeviceID,
		"qr_url":    fmt.Sprintf("/qr/%s", req.DeviceID),
		"status":    "waiting_for_qr_scan",
	})
}

// GetQRCode returns the QR code for a session
func GetQRCode(c *gin.Context) {
	deviceID := c.Param("device_id")

	waService := services.GetWhatsAppService()
	deviceClient, err := waService.GetSession(deviceID)
	if err != nil {
		// Check if request is from browser
		if isBrowserRequest(c) {
			renderHTMLError(c, "Session not found", "Please create session first via POST /session/create")
		} else {
			utils.ErrorResponse(c, http.StatusNotFound, "Session not found. Please create session first via POST /session/create")
		}
		return
	}

	// Check if already connected
	if deviceClient.Connected {
		if isBrowserRequest(c) {
			renderHTMLConnected(c, deviceID, deviceClient.Phone)
		} else {
			utils.SuccessResponse(c, http.StatusOK, "Already connected", gin.H{
				"device_id": deviceID,
				"status":    "connected",
				"phone":     deviceClient.Phone,
			})
		}
		return
	}

	// Check if client is properly initialized for QR code generation
	if deviceClient.Client == nil {
		if isBrowserRequest(c) {
			renderHTMLError(c, "Client not initialized", "Please recreate the session.")
		} else {
			utils.ErrorResponse(c, http.StatusInternalServerError, "Client not initialized. Please recreate the session.")
		}
		return
	}

	// Check if the client is already connected to WhatsApp servers
	if deviceClient.Client.IsLoggedIn() {
		// Client is logged in but not marked as connected in our state
		if deviceClient.Client.Store.ID != nil {
			deviceClient.Connected = true
			deviceClient.Phone = deviceClient.Client.Store.ID.User
			deviceClient.ConnectedAt = time.Now()

			if isBrowserRequest(c) {
				renderHTMLConnected(c, deviceID, deviceClient.Phone)
			} else {
				utils.SuccessResponse(c, http.StatusOK, "Already connected", gin.H{
					"device_id": deviceID,
					"status":    "connected",
					"phone":     deviceClient.Phone,
				})
			}
			return
		}
	}

	// Try to get QR code with shorter timeout first
	timeout1 := 5 * time.Second
	timeout2 := 15 * time.Second

	// First attempt - short timeout
	select {
	case qrCode := <-deviceClient.QRChan:
		if qrCode != "" {
			if isBrowserRequest(c) {
				renderHTMLQRCode(c, deviceID, qrCode)
			} else {
				utils.SuccessResponse(c, http.StatusOK, "QR code generated", gin.H{
					"device_id":  deviceID,
					"qr_code":    qrCode,
					"expires_in": 60,
				})
			}
			return
		}
	case <-time.After(timeout1):
		// If we have a client but no QR code yet, try longer timeout
		if deviceClient.Client.Store.ID == nil {
			select {
			case qrCode := <-deviceClient.QRChan:
				if qrCode != "" {
					if isBrowserRequest(c) {
						renderHTMLQRCode(c, deviceID, qrCode)
					} else {
						utils.SuccessResponse(c, http.StatusOK, "QR code generated", gin.H{
							"device_id":  deviceID,
							"qr_code":    qrCode,
							"expires_in": 60,
						})
					}
					return
				}
			case <-time.After(timeout2 - timeout1):
				// Check client state
				if deviceClient.Client.Store.ID == nil {
					if isBrowserRequest(c) {
						renderHTMLError(c, "QR code generation in progress", "QR code is being generated. This may take up to 30 seconds. Please refresh the page.")
					} else {
						utils.ErrorResponse(c, http.StatusRequestTimeout, "QR code generation in progress. Please try again in a few seconds.")
					}
				} else {
					// Client connected during waiting
					if isBrowserRequest(c) {
						renderHTMLConnected(c, deviceID, deviceClient.Client.Store.ID.User)
					} else {
						utils.SuccessResponse(c, http.StatusOK, "Connected", gin.H{
							"device_id": deviceID,
							"status":    "connected",
							"phone":     deviceClient.Client.Store.ID.User,
						})
					}
				}
				return
			}
		} else {
			// Client has ID but not connected in our state
			if isBrowserRequest(c) {
				renderHTMLConnected(c, deviceID, deviceClient.Client.Store.ID.User)
			} else {
				utils.SuccessResponse(c, http.StatusOK, "Connected", gin.H{
					"device_id": deviceID,
					"status":    "connected",
					"phone":     deviceClient.Client.Store.ID.User,
				})
			}
			return
		}
	}
}

// isBrowserRequest checks if the request is from a web browser
func isBrowserRequest(c *gin.Context) bool {
	accept := c.GetHeader("Accept")
	userAgent := c.GetHeader("User-Agent")

	// Check if Accept header contains text/html
	if strings.Contains(accept, "text/html") {
		return true
	}

	// Check common browser user agents
	browserAgents := []string{"Mozilla", "Chrome", "Safari", "Edge", "Opera", "Firefox"}
	for _, agent := range browserAgents {
		if strings.Contains(userAgent, agent) {
			return true
		}
	}

	return false
}

// renderHTMLQRCode renders HTML page with QR code
func renderHTMLQRCode(c *gin.Context, deviceID, qrCode string) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WAKU - WhatsApp QR Code</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 40px;
            max-width: 500px;
            width: 100%;
            text-align: center;
        }
        .logo {
            font-size: 48px;
            margin-bottom: 10px;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 28px;
        }
        .subtitle {
            color: #666;
            margin-bottom: 30px;
            font-size: 14px;
        }
        .device-id {
            background: #f0f0f0;
            padding: 10px 20px;
            border-radius: 10px;
            margin-bottom: 30px;
            font-family: monospace;
            color: #555;
        }
        .qr-container {
            background: white;
            padding: 20px;
            border-radius: 15px;
            border: 3px solid #667eea;
            margin-bottom: 30px;
            display: inline-block;
        }
        #qrcode {
            margin: 0 auto;
        }
        .instructions {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
            margin-bottom: 20px;
            text-align: left;
        }
        .instructions h3 {
            color: #333;
            margin-bottom: 15px;
            font-size: 16px;
        }
        .instructions ol {
            margin-left: 20px;
            color: #666;
            line-height: 1.8;
        }
        .instructions li {
            margin-bottom: 8px;
        }
        .status {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 10px;
            color: #667eea;
            font-weight: 500;
            margin-top: 20px;
        }
        .spinner {
            width: 20px;
            height: 20px;
            border: 3px solid #f3f3f3;
            border-top: 3px solid #667eea;
            border-radius: 50%;
            animation: spin 1s linear infinite;
        }
        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
        .refresh-btn {
            background: #667eea;
            color: white;
            border: none;
            padding: 12px 30px;
            border-radius: 10px;
            font-size: 16px;
            cursor: pointer;
            transition: background 0.3s;
            margin-top: 20px;
        }
        .refresh-btn:hover {
            background: #5568d3;
        }
        .footer {
            margin-top: 30px;
            color: #999;
            font-size: 12px;
        }
    </style>
    <script src="https://cdn.jsdelivr.net/npm/qrcode@1.5.3/build/qrcode.min.js"></script>
</head>
<body>
    <div class="container">
        <div class="logo">📱</div>
        <h1>WAKU WhatsApp API</h1>
        <p class="subtitle">Scan QR Code to Connect</p>

        <div class="device-id">
            Device ID: <strong>` + deviceID + `</strong>
        </div>

        <div class="qr-container">
            <canvas id="qrcode"></canvas>
        </div>

        <div class="instructions">
            <h3>📋 How to Connect:</h3>
            <ol>
                <li>Open <strong>WhatsApp</strong> on your phone</li>
                <li>Tap <strong>Menu</strong> or <strong>Settings</strong></li>
                <li>Tap <strong>Linked Devices</strong></li>
                <li>Tap <strong>Link a Device</strong></li>
                <li>Point your phone at this screen to scan the QR code</li>
            </ol>
        </div>

        <div class="status">
            <div class="spinner"></div>
            <span>Waiting for scan...</span>
        </div>

        <button class="refresh-btn" onclick="location.reload()">🔄 Refresh QR Code</button>

        <div class="footer">
            QR Code expires in 60 seconds
        </div>
    </div>

    <script>
        // Wait for QRCode library to load
        function initQRCode() {
            if (typeof QRCode === 'undefined') {
                setTimeout(initQRCode, 100);
                return;
            }

            // Generate QR Code
            const qrCode = '` + qrCode + `';
            const canvas = document.getElementById('qrcode');

            QRCode.toCanvas(canvas, qrCode, {
                width: 280,
                margin: 2,
                color: {
                    dark: '#000000',
                    light: '#ffffff'
                }
            }, function (error) {
                if (error) {
                    console.error('QR Code generation error:', error);
                    document.querySelector('.qr-container').innerHTML = '<p style="color: red;">Error generating QR code. Please refresh.</p>';
                }
            });
        }

        // Initialize QR code when page loads
        if (document.readyState === 'loading') {
            document.addEventListener('DOMContentLoaded', initQRCode);
        } else {
            initQRCode();
        }

        // Auto refresh after 55 seconds
        setTimeout(() => {
            location.reload();
        }, 55000);

        // Check connection status every 3 seconds
        let checkInterval = setInterval(async () => {
            try {
                const response = await fetch('/session/` + deviceID + `/status');
                const data = await response.json();

                if (data.data && data.data.connected) {
                    clearInterval(checkInterval);
                    document.querySelector('.status').innerHTML = '<span style="color: #28a745;">✅ Connected Successfully!</span>';
                    setTimeout(() => {
                        window.location.href = '/qr/` + deviceID + `';
                    }, 2000);
                }
            } catch (error) {
                console.error('Error checking status:', error);
            }
        }, 3000);
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// renderHTMLConnected renders HTML page for already connected session
func renderHTMLConnected(c *gin.Context, deviceID, phone string) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WAKU - Connected</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #11998e 0%, #38ef7d 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 40px;
            max-width: 500px;
            width: 100%;
            text-align: center;
        }
        .success-icon {
            font-size: 80px;
            margin-bottom: 20px;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 28px;
        }
        .subtitle {
            color: #666;
            margin-bottom: 30px;
        }
        .info-box {
            background: #f8f9fa;
            padding: 20px;
            border-radius: 10px;
            margin-bottom: 20px;
        }
        .info-row {
            display: flex;
            justify-content: space-between;
            padding: 10px 0;
            border-bottom: 1px solid #e0e0e0;
        }
        .info-row:last-child {
            border-bottom: none;
        }
        .info-label {
            color: #666;
            font-weight: 500;
        }
        .info-value {
            color: #333;
            font-family: monospace;
        }
        .btn {
            background: #11998e;
            color: white;
            border: none;
            padding: 12px 30px;
            border-radius: 10px;
            font-size: 16px;
            cursor: pointer;
            transition: background 0.3s;
            margin: 10px;
            text-decoration: none;
            display: inline-block;
        }
        .btn:hover {
            background: #0d7a6f;
        }
        .btn-danger {
            background: #dc3545;
        }
        .btn-danger:hover {
            background: #c82333;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="success-icon">✅</div>
        <h1>Already Connected!</h1>
        <p class="subtitle">Your WhatsApp session is active</p>

        <div class="info-box">
            <div class="info-row">
                <span class="info-label">Device ID:</span>
                <span class="info-value">` + deviceID + `</span>
            </div>
            <div class="info-row">
                <span class="info-label">Phone Number:</span>
                <span class="info-value">` + phone + `</span>
            </div>
            <div class="info-row">
                <span class="info-label">Status:</span>
                <span class="info-value" style="color: #28a745;">Connected</span>
            </div>
        </div>

        <a href="/session/` + deviceID + `/status" class="btn">📊 View Status</a>
        <button onclick="logout()" class="btn btn-danger">🚪 Logout</button>
    </div>

    <script>
        async function logout() {
            if (confirm('Are you sure you want to logout this session?')) {
                try {
                    const response = await fetch('/logout/` + deviceID + `', {
                        method: 'POST'
                    });
                    if (response.ok) {
                        alert('Logged out successfully!');
                        location.reload();
                    }
                } catch (error) {
                    alert('Error logging out: ' + error.message);
                }
            }
        }
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// renderHTMLError renders HTML error page
func renderHTMLError(c *gin.Context, title, message string) {
	html := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>WAKU - Error</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            padding: 20px;
        }
        .container {
            background: white;
            border-radius: 20px;
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.3);
            padding: 40px;
            max-width: 500px;
            width: 100%;
            text-align: center;
        }
        .error-icon {
            font-size: 80px;
            margin-bottom: 20px;
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
            font-size: 28px;
        }
        .message {
            color: #666;
            margin-bottom: 30px;
            line-height: 1.6;
        }
        .btn {
            background: #f5576c;
            color: white;
            border: none;
            padding: 12px 30px;
            border-radius: 10px;
            font-size: 16px;
            cursor: pointer;
            transition: background 0.3s;
            text-decoration: none;
            display: inline-block;
        }
        .btn:hover {
            background: #e04455;
        }
    </style>
</head>
<body>
    <div class="container">
        <div class="error-icon">❌</div>
        <h1>` + title + `</h1>
        <p class="message">` + message + `</p>
        <button onclick="location.reload()" class="btn">🔄 Try Again</button>
    </div>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// LogoutSession logs out a session
func LogoutSession(c *gin.Context) {
	deviceID := c.Param("device_id")

	waService := services.GetWhatsAppService()
	err := waService.Logout(deviceID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Session logged out successfully", gin.H{
		"device_id": deviceID,
		"status":    "disconnected",
	})
}

// DeleteSession deletes a session completely
func DeleteSession(c *gin.Context) {
	deviceID := c.Param("device_id")

	waService := services.GetWhatsAppService()
	err := waService.DeleteSession(deviceID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusNotFound, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Session deleted successfully", gin.H{
		"device_id":  deviceID,
		"deleted_at": time.Now().Format(time.RFC3339),
	})
}

// GetSessionStatus returns the status of a session
func GetSessionStatus(c *gin.Context) {
	deviceID := c.Param("device_id")

	waService := services.GetWhatsAppService()
	deviceClient, err := waService.GetSession(deviceID)
	if err != nil {
		utils.SuccessResponse(c, http.StatusOK, "Session status retrieved", gin.H{
			"device_id": deviceID,
			"status":    "not_found",
			"phone":     nil,
		})
		return
	}

	status := "disconnected"
	if deviceClient.Connected {
		status = "connected"
	} else if deviceClient.Client.Store.ID == nil {
		status = "waiting_for_qr_scan"
	}

	data := gin.H{
		"device_id": deviceID,
		"status":    status,
		"phone":     deviceClient.Phone,
	}

	if deviceClient.Connected {
		data["connected_at"] = deviceClient.ConnectedAt.Format(time.RFC3339)
	}

	utils.SuccessResponse(c, http.StatusOK, "Session status retrieved", data)
}

// ListSessions returns all sessions
func ListSessions(c *gin.Context) {
	waService := services.GetWhatsAppService()
	sessions := waService.GetAllSessions()

	sessionList := make([]gin.H, 0)
	for _, session := range sessions {
		status := "disconnected"
		if session.Connected {
			status = "connected"
		} else if session.Client.Store.ID == nil {
			status = "waiting_for_qr_scan"
		}

		sessionList = append(sessionList, gin.H{
			"device_id": session.DeviceID,
			"status":    status,
			"phone":     session.Phone,
		})
	}

	utils.SuccessResponse(c, http.StatusOK, "Sessions retrieved", gin.H{
		"total":    len(sessionList),
		"sessions": sessionList,
	})
}
