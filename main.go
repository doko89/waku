package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"waku/handlers"
	"waku/middleware"
	"waku/services"
	"waku/utils"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: .env file not found, using system environment variables")
	}

	// Ensure required directories exist
	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./sessions"
	}
	if err := utils.EnsureDir(sessionDir); err != nil {
		log.Fatalf("Failed to create session directory: %v", err)
	}

	tempDir := os.Getenv("TEMP_MEDIA_DIR")
	if tempDir == "" {
		tempDir = "./temp"
	}
	if err := utils.EnsureDir(tempDir); err != nil {
		log.Fatalf("Failed to create temp directory: %v", err)
	}

	// Initialize webhook service
	services.InitWebhookService()

	// Set Gin mode
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create Gin router
	router := gin.Default()

	// Add CORS middleware for all routes
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Public routes (no authentication required)
	router.GET("/qr/:device_id", handlers.GetQRCode)
	router.GET("/session/:device_id/status", handlers.GetSessionStatus) // Make status public for browser polling

	// Protected routes (require authentication)
	protected := router.Group("/")
	protected.Use(middleware.AuthMiddleware())
	{
		// Session management
		protected.POST("/session/create", handlers.CreateSession)
		protected.POST("/logout/:device_id", handlers.LogoutSession)
		protected.DELETE("/session/:device_id", handlers.DeleteSession)
		protected.GET("/sessions", handlers.ListSessions)

		// Messaging
		protected.POST("/send", handlers.SendMessage)
		protected.POST("/send-group", handlers.SendGroupMessage)

		// Media
		protected.POST("/send-media", handlers.SendMediaMessage)
		protected.POST("/send-group-media", handlers.SendGroupMediaMessage)

		// Information
		protected.GET("/contacts/:device_id", handlers.GetContacts)
		protected.GET("/groups/:device_id", handlers.GetGroups)
	}

	// Get port from environment
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	// Start server in a goroutine
	go func() {
		log.Printf("🚀 WAKU WhatsApp API Server starting on port %s", port)
		log.Printf("📝 API Documentation: http://localhost:%s", port)
		log.Printf("🔐 Authentication: Bearer token required (except /qr endpoint)")

		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("🛑 Shutting down server...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Disconnect all WhatsApp sessions
	waService := services.GetWhatsAppService()
	sessions := waService.GetAllSessions()
	for _, session := range sessions {
		log.Printf("Disconnecting session: %s", session.DeviceID)
		session.Client.Disconnect()
	}

	// Shutdown HTTP server
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("✅ Server exited gracefully")
}
