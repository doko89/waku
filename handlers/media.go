package handlers

import (
	"net/http"
	"os"
	"waku/services"
	"waku/utils"

	"github.com/gin-gonic/gin"
)

// SendMediaMessage sends a media message to a personal contact
func SendMediaMessage(c *gin.Context) {
	deviceID := c.PostForm("device_id")
	phone := c.PostForm("phone")
	caption := c.PostForm("caption")

	// Validate required fields
	if deviceID == "" || phone == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "device_id and phone are required")
		return
	}

	// Validate phone number format
	if len(phone) < 10 {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid phone number format. Use: country_code + number (e.g., 628123456789)")
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "File is required: "+err.Error())
		return
	}

	// Validate file size
	if err := utils.ValidateFileSize(file); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Save file to temp directory
	tempDir := os.Getenv("TEMP_MEDIA_DIR")
	if tempDir == "" {
		tempDir = "./temp"
	}

	filePath, err := utils.SaveUploadedFile(file, tempDir)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save file: "+err.Error())
		return
	}

	// Send media message
	waService := services.GetWhatsAppService()
	messageID, mediaType, fileSize, err := waService.SendMediaMessage(deviceID, phone, filePath, caption)

	// Delete temp file after sending
	defer utils.DeleteFile(filePath)

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Media sent successfully", gin.H{
		"message_id": messageID,
		"media_type": mediaType,
		"file_size":  fileSize,
	})
}

// SendGroupMediaMessage sends a media message to a group
func SendGroupMediaMessage(c *gin.Context) {
	deviceID := c.PostForm("device_id")
	groupJID := c.PostForm("group_jid")
	caption := c.PostForm("caption")

	// Validate required fields
	if deviceID == "" || groupJID == "" {
		utils.ErrorResponse(c, http.StatusBadRequest, "device_id and group_jid are required")
		return
	}

	// Validate group JID format
	if len(groupJID) < 10 || groupJID[len(groupJID)-5:] != "@g.us" {
		utils.ErrorResponse(c, http.StatusBadRequest, "Invalid group JID format. Should end with @g.us")
		return
	}

	// Get uploaded file
	file, err := c.FormFile("file")
	if err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, "File is required: "+err.Error())
		return
	}

	// Validate file size
	if err := utils.ValidateFileSize(file); err != nil {
		utils.ErrorResponse(c, http.StatusBadRequest, err.Error())
		return
	}

	// Save file to temp directory
	tempDir := os.Getenv("TEMP_MEDIA_DIR")
	if tempDir == "" {
		tempDir = "./temp"
	}

	filePath, err := utils.SaveUploadedFile(file, tempDir)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, "Failed to save file: "+err.Error())
		return
	}

	// Send media message
	waService := services.GetWhatsAppService()
	messageID, mediaType, fileSize, err := waService.SendGroupMediaMessage(deviceID, groupJID, filePath, caption)

	// Delete temp file after sending
	defer utils.DeleteFile(filePath)

	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Group media sent successfully", gin.H{
		"message_id": messageID,
		"media_type": mediaType,
		"file_size":  fileSize,
	})
}
