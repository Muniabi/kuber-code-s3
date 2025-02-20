package handler

import (
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"path/filepath"
	"strings"

	"kuber-code-s3/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// @title File Storage Service API
// @version 1.0
// @description Microservice for file storage with Minio and MongoDB
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@filestorage.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey ApiKeyAuth
// @in header
// @name Authorization

type FileHandler struct {
	service *service.FileService
}

type SuccessResponse struct {
	URL string `json:"url"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

// NewFileHandler creates a new file handler
func NewFileHandler(service *service.FileService) *FileHandler {
	return &FileHandler{service: service}
}

// UploadFile godoc
// @Summary Upload a file
// @Description Upload file to storage
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param file formData file true "File to upload"
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/upload [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	// Validate file size
	const maxUploadSize = 1024 << 20 // 1024 MB = 1 GB
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadSize)

	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("File upload error: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "File upload error"})
		return
	}

	// Log file info
	log.Printf("Upload attempt: Filename=%s, Size=%d, MIME=%s",
		file.Filename, file.Size, file.Header.Get("Content-Type"))

	// Validate file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".mp4":  true,
		".mov":  true,
		".avi":  true,
		".mkv":  true,
	}
	if !allowedExtensions[ext] {
		log.Printf("Unsupported file extension: %s", ext)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Unsupported file extension"})
		return
	}

	// Detect real content type
	contentType, err := detectContentType(file)
	if err != nil {
		log.Printf("Content type detection error: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid file content"})
		return
	}

	// Validate content type
	allowedTypes := map[string]bool{
		"image/jpeg":      true,
		"image/png":       true,
		"video/mp4":       true,
		"video/quicktime": true,
		"video/x-msvideo": true,
		"video/x-matroska": true,
	}
	if !allowedTypes[contentType] {
		log.Printf("Unsupported content type: %s", contentType)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Unsupported file type"})
		return
	}

	// Upload file
	url, err := h.service.UploadFile(c.Request.Context(), file)
	if err != nil {
		log.Printf("File upload service error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to process file"})
		return
	}

	log.Printf("File uploaded successfully: %s", url)
	c.JSON(http.StatusOK, SuccessResponse{URL: url})
}

// DeleteFile godoc
// @Summary Delete a file
// @Description Delete file from storage
// @Tags files
// @Produce json
// @Param id path string true "File ID"
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/files/{id} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	fileID := c.Param("id")

	if _, err := uuid.Parse(fileID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid file ID format"})
		return
	}

	err := h.service.DeleteFile(c.Request.Context(), fileID)
	if err != nil {
		if err == service.ErrFileNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "File not found"})
			return
		}
		log.Printf("File deletion error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{URL: fmt.Sprintf("File %s deleted", fileID)})
}

// ReplaceFile godoc
// @Summary Replace a file
// @Description Replace existing file
// @Tags files
// @Accept multipart/form-data
// @Produce json
// @Param id path string true "File ID"
// @Param file formData file true "New file"
// @Security ApiKeyAuth
// @Success 200 {object} SuccessResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/files/{id} [put]
func (h *FileHandler) ReplaceFile(c *gin.Context) {
	fileID := c.Param("id")

	if _, err := uuid.Parse(fileID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid file ID format"})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		log.Printf("File upload error: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "File upload error"})
		return
	}

	// Validate new file
	contentType, err := detectContentType(file)
	if err != nil {
		log.Printf("Content type detection error: %v", err)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid file content"})
		return
	}

	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/png":  true,
		"video/mp4":  true,
	}
	if !allowedTypes[contentType] {
		log.Printf("Unsupported content type: %s", contentType)
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Unsupported file type"})
		return
	}

	url, err := h.service.ReplaceFile(c.Request.Context(), fileID, file)
	if err != nil {
		if err == service.ErrFileNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "File not found"})
			return
		}
		log.Printf("File replacement error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to replace file"})
		return
	}

	c.JSON(http.StatusOK, SuccessResponse{URL: url})
}

// GetFileMetadata godoc
// @Summary Get file metadata
// @Description Get file metadata by ID
// @Tags files
// @Produce json
// @Param id path string true "File ID"
// @Security ApiKeyAuth
// @Success 200 {object} models.FileMetadata
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/files/{id} [get]
func (h *FileHandler) GetFileMetadata(c *gin.Context) {
	fileID := c.Param("id")

	if _, err := uuid.Parse(fileID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid file ID format"})
		return
	}

	metadata, err := h.service.GetFileMetadata(c.Request.Context(), fileID)
	if err != nil {
		if err == service.ErrFileNotFound {
			c.JSON(http.StatusNotFound, ErrorResponse{Error: "File not found"})
			return
		}
		log.Printf("Metadata retrieval error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get file metadata"})
		return
	}

	c.JSON(http.StatusOK, metadata)
}

// detectContentType detects the real content type of a file
func detectContentType(file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	buf := make([]byte, 512)
	if _, err = src.Read(buf); err != nil {
		return "", err
	}

	contentType := http.DetectContentType(buf)
	if _, err = src.Seek(0, 0); err != nil {
		return "", err
	}

	return contentType, nil
}