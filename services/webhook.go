package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.mau.fi/whatsmeow/types/events"
)

// WebhookPayload represents the data sent to webhook URL
type WebhookPayload struct {
	DeviceID        string      `json:"device_id"`
	MessageID       string      `json:"message_id"`
	From            string      `json:"from"`
	FromName        string      `json:"from_name"`
	Message         string      `json:"message"`
	MessageType     string      `json:"message_type"`
	Timestamp       int64       `json:"timestamp"`
	IsGroup         bool        `json:"is_group"`
	GroupJID        *string     `json:"group_jid"`
	GroupName       *string     `json:"group_name"`
	MediaURL        *string     `json:"media_url"`
	QuotedMessage   interface{} `json:"quoted_message"`
}

// WebhookService handles sending incoming messages to webhook URL
type WebhookService struct {
	enabled    bool
	webhookURL string
	retryCount int
	httpClient *http.Client
}

var webhookService *WebhookService

// InitWebhookService initializes the webhook service
func InitWebhookService() {
	enabled, _ := strconv.ParseBool(os.Getenv("WEBHOOK_ENABLED"))
	retryCount, _ := strconv.Atoi(os.Getenv("WEBHOOK_RETRY"))
	if retryCount == 0 {
		retryCount = 3
	}

	webhookService = &WebhookService{
		enabled:    enabled,
		webhookURL: os.Getenv("WEBHOOK_URL"),
		retryCount: retryCount,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetWebhookService returns the webhook service instance
func GetWebhookService() *WebhookService {
	if webhookService == nil {
		InitWebhookService()
	}
	return webhookService
}

// HandleIncomingMessage processes incoming WhatsApp messages and sends to webhook
func (w *WebhookService) HandleIncomingMessage(deviceID string, evt *events.Message) {
	if !w.enabled || w.webhookURL == "" {
		fmt.Printf("Webhook disabled or URL not set, skipping message forwarding\n")
		return
	}

	fmt.Printf("Forwarding message from device %s to webhook: %s\n", deviceID, w.webhookURL)

	// Build webhook payload
	payload := WebhookPayload{
		DeviceID:    deviceID,
		MessageID:   evt.Info.ID,
		From:        evt.Info.Sender.String(),
		FromName:    evt.Info.PushName,
		Timestamp:   evt.Info.Timestamp.Unix(),
		IsGroup:     evt.Info.IsGroup,
		MessageType: "text",
	}

	// Extract message content
	if evt.Message.GetConversation() != "" {
		payload.Message = evt.Message.GetConversation()
	} else if evt.Message.GetExtendedTextMessage() != nil {
		payload.Message = evt.Message.GetExtendedTextMessage().GetText()
	} else if evt.Message.GetImageMessage() != nil {
		payload.MessageType = "image"
		payload.Message = evt.Message.GetImageMessage().GetCaption()
	} else if evt.Message.GetVideoMessage() != nil {
		payload.MessageType = "video"
		payload.Message = evt.Message.GetVideoMessage().GetCaption()
	} else if evt.Message.GetAudioMessage() != nil {
		payload.MessageType = "audio"
	} else if evt.Message.GetDocumentMessage() != nil {
		payload.MessageType = "document"
		payload.Message = evt.Message.GetDocumentMessage().GetCaption()
	}

	// Handle group messages
	if evt.Info.IsGroup {
		groupJID := evt.Info.Chat.String()
		payload.GroupJID = &groupJID
		// Group name would need to be fetched from group info
	}

	// Send to webhook with retry
	fmt.Printf("Sending webhook payload: %+v\n", payload)
	go w.sendWithRetry(payload)
}

// sendWithRetry sends payload to webhook with exponential backoff retry
func (w *WebhookService) sendWithRetry(payload WebhookPayload) {
	var lastErr error
	
	for attempt := 0; attempt < w.retryCount; attempt++ {
		fmt.Printf("Webhook attempt %d/%d\n", attempt+1, w.retryCount)
		err := w.send(payload)
		if err == nil {
			fmt.Printf("Webhook sent successfully on attempt %d\n", attempt+1)
			return // Success
		}

		lastErr = err
		fmt.Printf("Webhook attempt %d failed: %v\n", attempt+1, err)

		// Exponential backoff
		if attempt < w.retryCount-1 {
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			fmt.Printf("Retrying webhook in %v\n", backoff)
			time.Sleep(backoff)
		}
	}

	// Log final failure
	fmt.Printf("Failed to send webhook after %d attempts: %v\n", w.retryCount, lastErr)
}

// send sends the payload to webhook URL
func (w *WebhookService) send(payload WebhookPayload) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", w.webhookURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status code: %d", resp.StatusCode)
	}

	return nil
}

// SetupEventHandler sets up the event handler for a device client
func SetupEventHandler(deviceID string, dc *DeviceClient) {
	webhookSvc := GetWebhookService()
	
	dc.EventHandler = func(evt interface{}) {
		switch v := evt.(type) {
		case *events.Message:
			webhookSvc.HandleIncomingMessage(deviceID, v)
		}
	}
}

