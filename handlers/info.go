package handlers

import (
	"net/http"
	"waku/services"
	"waku/utils"

	"github.com/gin-gonic/gin"
)

// GetContacts retrieves the contact list for a device
func GetContacts(c *gin.Context) {
	deviceID := c.Param("device_id")

	waService := services.GetWhatsAppService()
	contacts, err := waService.GetContacts(deviceID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Contacts retrieved", gin.H{
		"total":    len(contacts),
		"contacts": contacts,
	})
}

// GetGroups retrieves the group list for a device
func GetGroups(c *gin.Context) {
	deviceID := c.Param("device_id")

	waService := services.GetWhatsAppService()
	groups, err := waService.GetGroups(deviceID)
	if err != nil {
		utils.ErrorResponse(c, http.StatusInternalServerError, err.Error())
		return
	}

	utils.SuccessResponse(c, http.StatusOK, "Groups retrieved", gin.H{
		"total":  len(groups),
		"groups": groups,
	})
}

