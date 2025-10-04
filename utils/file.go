package utils

import (
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
)

// MediaType represents the type of media file
type MediaType string

const (
	MediaTypeImage    MediaType = "image"
	MediaTypeVideo    MediaType = "video"
	MediaTypeAudio    MediaType = "audio"
	MediaTypeDocument MediaType = "document"
)

// MediaLimits defines size limits for different media types (in bytes)
var MediaLimits = map[MediaType]int64{
	MediaTypeImage:    16 * 1024 * 1024,  // 16MB
	MediaTypeVideo:    64 * 1024 * 1024,  // 64MB
	MediaTypeAudio:    16 * 1024 * 1024,  // 16MB
	MediaTypeDocument: 100 * 1024 * 1024, // 100MB
}

// GetMediaType determines the media type based on file extension
func GetMediaType(filename string) MediaType {
	ext := strings.ToLower(filepath.Ext(filename))
	
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif"}
	videoExts := []string{".mp4", ".avi", ".mkv"}
	audioExts := []string{".mp3", ".ogg", ".m4a"}
	
	for _, e := range imageExts {
		if ext == e {
			return MediaTypeImage
		}
	}
	
	for _, e := range videoExts {
		if ext == e {
			return MediaTypeVideo
		}
	}
	
	for _, e := range audioExts {
		if ext == e {
			return MediaTypeAudio
		}
	}
	
	return MediaTypeDocument
}

// ValidateFileSize checks if file size is within limits for its type
func ValidateFileSize(fileHeader *multipart.FileHeader) error {
	mediaType := GetMediaType(fileHeader.Filename)
	maxSize := MediaLimits[mediaType]
	
	if fileHeader.Size > maxSize {
		return fmt.Errorf("file size exceeds maximum limit for %s type (%d MB)", mediaType, maxSize/(1024*1024))
	}
	
	return nil
}

// SaveUploadedFile saves an uploaded file to the specified directory
func SaveUploadedFile(fileHeader *multipart.FileHeader, destDir string) (string, error) {
	// Create destination directory if it doesn't exist
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %v", err)
	}
	
	// Generate destination path
	destPath := filepath.Join(destDir, fileHeader.Filename)
	
	// Open source file
	src, err := fileHeader.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open uploaded file: %v", err)
	}
	defer src.Close()
	
	// Create destination file
	dst, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dst.Close()
	
	// Copy file content
	if _, err := dst.ReadFrom(src); err != nil {
		return "", fmt.Errorf("failed to save file: %v", err)
	}
	
	return destPath, nil
}

// DeleteFile removes a file from the filesystem
func DeleteFile(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}
	return nil
}

// EnsureDir creates a directory if it doesn't exist
func EnsureDir(dirPath string) error {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %v", err)
	}
	return nil
}

