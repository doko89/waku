package handlers

import (
	"fmt"
	"net/http"
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
		utils.ErrorResponse(c, http.StatusNotFound, "Session not found. Please create session first via POST /session/create")
		return
	}

	// Check if already connected
	if deviceClient.Connected {
		utils.SuccessResponse(c, http.StatusOK, "Already connected", gin.H{
			"device_id": deviceID,
			"status":    "connected",
			"phone":     deviceClient.Phone,
		})
		return
	}

	// Wait for QR code with timeout
	select {
	case qrCode := <-deviceClient.QRChan:
		utils.SuccessResponse(c, http.StatusOK, "QR code generated", gin.H{
			"device_id":  deviceID,
			"qr_code":    qrCode,
			"expires_in": 60,
		})

	case <-time.After(5 * time.Second):
		utils.ErrorResponse(c, http.StatusRequestTimeout, "QR code not ready yet. Please try again.")
	}
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
