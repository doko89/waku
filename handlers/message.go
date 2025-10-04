package handlers

import (
	"net/http"
	"waku/services"
	"waku/utils"

	"github.com/gin-gonic/gin"
)

// SendMessageRequest represents the request body for sending a message
type SendMessageRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	Phone    string `json:"phone" binding:"required"`
	Message  string `json:"message" binding:"required"`
}

// SendGroupMessageRequest represents the request body for sending a group message
type SendGroupMessageRequest struct {
	DeviceID string `json:"device_id" binding:"required"`
	GroupJID string `json:"group_jid" binding:"required"`
	Message  string `json:"message" binding:"required"`
}

// SendMessage sends a personal message
func SendMessage(c *gin.Context) {
	var req SendMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate phone number format
	if len(req.Phone) < 10 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid phone number format. Use: country_code + number (e.g., 628123456789)")
		return
	}

	waService := services.GetWhatsAppService()
	messageID, timestamp, err := waService.SendMessage(req.DeviceID, req.Phone, req.Message)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Message sent successfully", gin.H{
		"message_id": messageID,
		"timestamp":  timestamp,
	})
}

// SendGroupMessage sends a group message
func SendGroupMessage(c *gin.Context) {
	var req SendGroupMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid request: "+err.Error())
		return
	}

	// Validate group JID format
	if len(req.GroupJID) < 10 || req.GroupJID[len(req.GroupJID)-5:] != "@g.us" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid group JID format. Should end with @g.us")
		return
	}

	waService := services.GetWhatsAppService()
	messageID, timestamp, err := waService.SendGroupMessage(req.DeviceID, req.GroupJID, req.Message)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Group message sent successfully", gin.H{
		"message_id": messageID,
		"timestamp":  timestamp,
	})
}

