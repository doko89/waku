package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	waProto "go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"

	_ "modernc.org/sqlite"
)

// DeviceClient represents a WhatsApp client for a specific device
type DeviceClient struct {
	Client       *whatsmeow.Client
	DeviceID     string
	QRChan       chan string
	Connected    bool
	Phone        string
	ConnectedAt  time.Time
	EventHandler func(interface{})
}

// WhatsAppService manages multiple WhatsApp device clients
type WhatsAppService struct {
	clients map[string]*DeviceClient
	mu      sync.RWMutex
	logger  waLog.Logger
}

var (
	waService     *WhatsAppService
	waServiceOnce sync.Once
)

// GetWhatsAppService returns the singleton instance of WhatsAppService
func GetWhatsAppService() *WhatsAppService {
	waServiceOnce.Do(func() {
		waService = &WhatsAppService{
			clients: make(map[string]*DeviceClient),
			logger:  waLog.Stdout("WhatsApp", "INFO", true),
		}

		// Load existing sessions from disk
		if err := waService.loadExistingSessions(); err != nil {
			waService.logger.Errorf("Failed to load existing sessions: %v", err)
		}
	})
	return waService
}

// loadExistingSessions loads all existing sessions from SESSION_DIR
func (s *WhatsAppService) loadExistingSessions() error {
	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./sessions"
	}

	// Check if session directory exists
	if _, err := os.Stat(sessionDir); os.IsNotExist(err) {
		s.logger.Infof("Session directory does not exist, skipping session loading")
		return nil
	}

	// Read all directories in session directory
	entries, err := os.ReadDir(sessionDir)
	if err != nil {
		return fmt.Errorf("failed to read session directory: %v", err)
	}

	loadedCount := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		deviceID := entry.Name()
		dbPath := filepath.Join(sessionDir, deviceID, "session.db")

		// Check if session database exists
		if _, err := os.Stat(dbPath); os.IsNotExist(err) {
			s.logger.Warnf("Session database not found for device %s, skipping", deviceID)
			continue
		}

		// Load the session
		if err := s.loadSession(deviceID); err != nil {
			s.logger.Errorf("Failed to load session for device %s: %v", deviceID, err)
			continue
		}

		loadedCount++
		s.logger.Infof("Loaded existing session for device: %s", deviceID)
	}

	if loadedCount > 0 {
		s.logger.Infof("Successfully loaded %d existing session(s)", loadedCount)
	} else {
		s.logger.Infof("No existing sessions found")
	}

	return nil
}

// loadSession loads a single session from disk
func (s *WhatsAppService) loadSession(deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session already loaded
	if _, exists := s.clients[deviceID]; exists {
		return fmt.Errorf("session already loaded for device_id: %s", deviceID)
	}

	// Get session directory
	sessionDir := filepath.Join(os.Getenv("SESSION_DIR"), deviceID)
	dbPath := filepath.Join(sessionDir, "session.db")

	// Create database container
	container, err := sqlstore.New(context.Background(), "sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", dbPath), s.logger)
	if err != nil {
		return fmt.Errorf("failed to open database: %v", err)
	}

	// Get first device from store
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get device from store: %v", err)
	}

	if deviceStore == nil {
		return fmt.Errorf("no device found in database")
	}

	// Create WhatsApp client
	client := whatsmeow.NewClient(deviceStore, s.logger)

	// Create device client
	deviceClient := &DeviceClient{
		Client:   client,
		DeviceID: deviceID,
		QRChan:   make(chan string, 5),
	}

	// Set event handler
	client.AddEventHandler(deviceClient.eventHandler)

	// Connect to WhatsApp
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %v", err)
	}

	// Check if already logged in
	if client.IsLoggedIn() {
		deviceClient.Connected = true
		deviceClient.Phone = client.Store.ID.User
		deviceClient.ConnectedAt = time.Now()
		s.logger.Infof("Device %s is already logged in as %s", deviceID, deviceClient.Phone)
	} else {
		s.logger.Infof("Device %s loaded but not logged in, waiting for QR scan", deviceID)
	}

	// Store client
	s.clients[deviceID] = deviceClient

	return nil
}

// CreateSession creates a new WhatsApp session for a device
func (s *WhatsAppService) CreateSession(deviceID string) (*DeviceClient, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if session already exists
	if client, exists := s.clients[deviceID]; exists {
		return client, fmt.Errorf("session already exists for device_id: %s", deviceID)
	}

	// Create session directory
	sessionDir := filepath.Join(os.Getenv("SESSION_DIR"), deviceID)
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create session directory: %v", err)
	}

	// Create database container
	dbPath := filepath.Join(sessionDir, "session.db")
	container, err := sqlstore.New(context.Background(), "sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", dbPath), s.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create database: %v", err)
	}

	// Get first device from container
	deviceStore, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get device: %v", err)
	}

	// Create WhatsApp client
	client := whatsmeow.NewClient(deviceStore, s.logger)

	// Create device client
	deviceClient := &DeviceClient{
		Client:    client,
		DeviceID:  deviceID,
		QRChan:    make(chan string, 5),
		Connected: false,
	}

	// Set up event handler
	client.AddEventHandler(deviceClient.eventHandler)

	// Store client
	s.clients[deviceID] = deviceClient

	// Connect client
	if client.Store.ID == nil {
		// No session exists, need to pair with QR
		go s.connectWithQR(deviceClient)
	} else {
		// Session exists, try to reconnect
		go s.reconnect(deviceClient)
	}

	return deviceClient, nil
}

// connectWithQR connects a client using QR code pairing
func (s *WhatsAppService) connectWithQR(dc *DeviceClient) {
	// Event handler is already set up, just connect
	err := dc.Client.Connect()
	if err != nil {
		s.logger.Errorf("Failed to connect: %v", err)
		return
	}

	s.logger.Infof("QR code connection initiated for device: %s", dc.DeviceID)
}

// reconnect attempts to reconnect an existing session
func (s *WhatsAppService) reconnect(dc *DeviceClient) {
	err := dc.Client.Connect()
	if err != nil {
		s.logger.Errorf("Failed to reconnect device %s: %v", dc.DeviceID, err)
		return
	}
	s.logger.Infof("Device %s reconnected successfully", dc.DeviceID)
}

// eventHandler handles WhatsApp events for a device
func (dc *DeviceClient) eventHandler(evt interface{}) {
	switch v := evt.(type) {
	case *events.QR:
		// QR code event - send all codes to channel
		for _, code := range v.Codes {
			select {
			case dc.QRChan <- code:
				// Successfully sent QR code
			default:
				// Channel is full, clear old QR codes and try again
				select {
				case <-dc.QRChan:
					dc.QRChan <- code
				default:
					// Still can't send, skip this QR code
				}
			}
		}

	case *events.PairSuccess:
		// QR code scanned successfully
		// Note: logger access will be fixed by making logger available to device client

	case *events.Connected:
		dc.Connected = true
		dc.ConnectedAt = time.Now()
		if dc.Client.Store.ID != nil {
			dc.Phone = dc.Client.Store.ID.User
		}

	case *events.Disconnected:
		dc.Connected = false

	case *events.Message:
		// Handle incoming message - will be processed by webhook service
		if dc.EventHandler != nil {
			dc.EventHandler(v)
		}
	}
}

// GetSession retrieves a session by device ID
func (s *WhatsAppService) GetSession(deviceID string) (*DeviceClient, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[deviceID]
	if !exists {
		return nil, fmt.Errorf("session not found for device_id: %s", deviceID)
	}

	return client, nil
}

// GetAllSessions returns all active sessions
func (s *WhatsAppService) GetAllSessions() []*DeviceClient {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sessions := make([]*DeviceClient, 0, len(s.clients))
	for _, client := range s.clients {
		sessions = append(sessions, client)
	}

	return sessions
}

// Logout disconnects a session but keeps the session data
func (s *WhatsAppService) Logout(deviceID string) error {
	s.mu.RLock()
	client, exists := s.clients[deviceID]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("session not found for device_id: %s", deviceID)
	}

	client.Client.Disconnect()
	client.Connected = false

	return nil
}

// DeleteSession removes a session completely
func (s *WhatsAppService) DeleteSession(deviceID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	client, exists := s.clients[deviceID]
	if !exists {
		return fmt.Errorf("session not found for device_id: %s", deviceID)
	}

	// Disconnect client
	client.Client.Disconnect()

	// Remove from map
	delete(s.clients, deviceID)

	// Delete session directory
	sessionDir := filepath.Join(os.Getenv("SESSION_DIR"), deviceID)
	if err := os.RemoveAll(sessionDir); err != nil {
		return fmt.Errorf("failed to delete session directory: %v", err)
	}

	return nil
}

// SendMessage sends a text message to a phone number
func (s *WhatsAppService) SendMessage(deviceID, phone, message string) (string, int64, error) {
	client, err := s.GetSession(deviceID)
	if err != nil {
		return "", 0, err
	}

	if !client.Connected {
		return "", 0, fmt.Errorf("session not connected. Please scan QR code first")
	}

	// Parse JID
	jid := types.NewJID(phone, types.DefaultUserServer)

	// Send message
	msg := &waProto.Message{
		Conversation: &message,
	}

	resp, err := client.Client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send message: %v", err)
	}

	return resp.ID, resp.Timestamp.Unix(), nil
}

// SendGroupMessage sends a text message to a group
func (s *WhatsAppService) SendGroupMessage(deviceID, groupJID, message string) (string, int64, error) {
	client, err := s.GetSession(deviceID)
	if err != nil {
		return "", 0, err
	}

	if !client.Connected {
		return "", 0, fmt.Errorf("session not connected. Please scan QR code first")
	}

	// Parse group JID
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return "", 0, fmt.Errorf("invalid group JID: %v", err)
	}

	// Send message
	msg := &waProto.Message{
		Conversation: &message,
	}

	resp, err := client.Client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return "", 0, fmt.Errorf("failed to send group message: %v", err)
	}

	return resp.ID, resp.Timestamp.Unix(), nil
}

// SendMediaMessage sends a media message to a phone number
func (s *WhatsAppService) SendMediaMessage(deviceID, phone, filePath, caption string) (string, string, int64, error) {
	client, err := s.GetSession(deviceID)
	if err != nil {
		return "", "", 0, err
	}

	if !client.Connected {
		return "", "", 0, fmt.Errorf("session not connected. Please scan QR code first")
	}

	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to read file: %v", err)
	}

	// Upload media
	uploaded, err := client.Client.Upload(context.Background(), fileData, whatsmeow.MediaImage)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to upload media: %v", err)
	}

	// Parse JID
	jid := types.NewJID(phone, types.DefaultUserServer)

	// Determine media type and create message
	var msg *waProto.Message
	ext := filepath.Ext(filePath)
	mimetype := getMimeType(ext)
	fileLen := uint64(len(fileData))

	switch {
	case isImageExt(ext):
		msg = &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:       proto.String(caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(fileLen),
			},
		}
	case isVideoExt(ext):
		msg = &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Caption:       proto.String(caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(fileLen),
			},
		}
	case isAudioExt(ext):
		msg = &waProto.Message{
			AudioMessage: &waProto.AudioMessage{
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(fileLen),
			},
		}
	default:
		// Document
		msg = &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Caption:       proto.String(caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(fileLen),
				FileName:      proto.String(filepath.Base(filePath)),
			},
		}
	}

	// Send message
	resp, err := client.Client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to send media: %v", err)
	}

	mediaType := getMediaTypeString(ext)
	return resp.ID, mediaType, int64(fileLen), nil
}

// SendGroupMediaMessage sends a media message to a group
func (s *WhatsAppService) SendGroupMediaMessage(deviceID, groupJID, filePath, caption string) (string, string, int64, error) {
	client, err := s.GetSession(deviceID)
	if err != nil {
		return "", "", 0, err
	}

	if !client.Connected {
		return "", "", 0, fmt.Errorf("session not connected. Please scan QR code first")
	}

	// Read file
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to read file: %v", err)
	}

	// Upload media
	uploaded, err := client.Client.Upload(context.Background(), fileData, whatsmeow.MediaImage)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to upload media: %v", err)
	}

	// Parse group JID
	jid, err := types.ParseJID(groupJID)
	if err != nil {
		return "", "", 0, fmt.Errorf("invalid group JID: %v", err)
	}

	// Determine media type and create message
	var msg *waProto.Message
	ext := filepath.Ext(filePath)
	mimetype := getMimeType(ext)
	fileLen := uint64(len(fileData))

	switch {
	case isImageExt(ext):
		msg = &waProto.Message{
			ImageMessage: &waProto.ImageMessage{
				Caption:       proto.String(caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(fileLen),
			},
		}
	case isVideoExt(ext):
		msg = &waProto.Message{
			VideoMessage: &waProto.VideoMessage{
				Caption:       proto.String(caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(fileLen),
			},
		}
	default:
		msg = &waProto.Message{
			DocumentMessage: &waProto.DocumentMessage{
				Caption:       proto.String(caption),
				URL:           proto.String(uploaded.URL),
				DirectPath:    proto.String(uploaded.DirectPath),
				MediaKey:      uploaded.MediaKey,
				Mimetype:      proto.String(mimetype),
				FileEncSHA256: uploaded.FileEncSHA256,
				FileSHA256:    uploaded.FileSHA256,
				FileLength:    proto.Uint64(fileLen),
				FileName:      proto.String(filepath.Base(filePath)),
			},
		}
	}

	// Send message
	resp, err := client.Client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to send group media: %v", err)
	}

	mediaType := getMediaTypeString(ext)
	return resp.ID, mediaType, int64(fileLen), nil
}

// GetContacts retrieves the contact list for a device
func (s *WhatsAppService) GetContacts(deviceID string) ([]map[string]interface{}, error) {
	client, err := s.GetSession(deviceID)
	if err != nil {
		return nil, err
	}

	if !client.Connected {
		return nil, fmt.Errorf("session not connected. Please scan QR code first")
	}

	// Get contacts from store
	contacts, err := client.Client.Store.Contacts.GetAllContacts(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get contacts: %v", err)
	}

	// Format contacts
	result := make([]map[string]interface{}, 0)
	for jid, contact := range contacts {
		result = append(result, map[string]interface{}{
			"jid":         jid.String(),
			"name":        contact.FullName,
			"notify":      contact.PushName,
			"is_business": contact.BusinessName != "",
		})
	}

	return result, nil
}

// GetGroups retrieves the group list for a device
func (s *WhatsAppService) GetGroups(deviceID string) ([]map[string]interface{}, error) {
	client, err := s.GetSession(deviceID)
	if err != nil {
		return nil, err
	}

	if !client.Connected {
		return nil, fmt.Errorf("session not connected. Please scan QR code first")
	}

	// Get groups
	groups, err := client.Client.GetJoinedGroups(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get groups: %v", err)
	}

	// Format groups
	result := make([]map[string]interface{}, 0)
	for _, group := range groups {
		// Get group info
		groupInfo, err := client.Client.GetGroupInfo(group.JID)
		if err != nil {
			continue
		}

		result = append(result, map[string]interface{}{
			"jid":          group.JID.String(),
			"name":         groupInfo.Name,
			"participants": len(groupInfo.Participants),
			"is_admin":     isGroupAdmin(client.Client.Store.ID.ToNonAD(), groupInfo),
		})
	}

	return result, nil
}

// Helper functions

func isImageExt(ext string) bool {
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif"}
	for _, e := range imageExts {
		if strings.ToLower(ext) == e {
			return true
		}
	}
	return false
}

func isVideoExt(ext string) bool {
	videoExts := []string{".mp4", ".avi", ".mkv"}
	for _, e := range videoExts {
		if strings.ToLower(ext) == e {
			return true
		}
	}
	return false
}

func isAudioExt(ext string) bool {
	audioExts := []string{".mp3", ".ogg", ".m4a"}
	for _, e := range audioExts {
		if strings.ToLower(ext) == e {
			return true
		}
	}
	return false
}

func getMediaTypeString(ext string) string {
	switch {
	case isImageExt(ext):
		return "image"
	case isVideoExt(ext):
		return "video"
	case isAudioExt(ext):
		return "audio"
	default:
		return "document"
	}
}

func isGroupAdmin(userJID types.JID, groupInfo *types.GroupInfo) bool {
	for _, participant := range groupInfo.Participants {
		if participant.JID.User == userJID.User {
			return participant.IsAdmin || participant.IsSuperAdmin
		}
	}
	return false
}

func getMimeType(ext string) string {
	mimeTypes := map[string]string{
		".jpg":  "image/jpeg",
		".jpeg": "image/jpeg",
		".png":  "image/png",
		".gif":  "image/gif",
		".mp4":  "video/mp4",
		".avi":  "video/x-msvideo",
		".mkv":  "video/x-matroska",
		".mp3":  "audio/mpeg",
		".ogg":  "audio/ogg",
		".m4a":  "audio/mp4",
		".pdf":  "application/pdf",
		".doc":  "application/msword",
		".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		".xls":  "application/vnd.ms-excel",
		".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		".zip":  "application/zip",
	}

	if mime, ok := mimeTypes[strings.ToLower(ext)]; ok {
		return mime
	}
	return "application/octet-stream"
}
