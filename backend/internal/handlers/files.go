package handlers

import (
	"encoding/json"
	"fmt"
	"image"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sonar-annotation-backend/internal/config"
	"sonar-annotation-backend/internal/database"
	"sonar-annotation-backend/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type FileInfo struct {
	Width  int
	Height int
}

func ListSonarFiles(c *gin.Context) {
	var files []models.SonarFile
	if err := database.DB.Order("created_at DESC").Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, files)
}

func GetSonarFile(c *gin.Context) {
	id := c.Param("id")
	var file models.SonarFile
	if err := database.DB.First(&file, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	c.JSON(http.StatusOK, file)
}

func UploadSonarFile(c *gin.Context) {
	cfg := config.Load()

	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload directory"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No file uploaded"})
		return
	}
	defer file.Close()

	if header.Size > cfg.MaxUploadSize {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File too large"})
		return
	}

	ext := filepath.Ext(header.Filename)
	fileID := uuid.New().String()
	filename := fileID + ext
	filePath := filepath.Join(cfg.UploadDir, filename)

	out, err := os.Create(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create file"})
		return
	}
	defer out.Close()

	if _, err := io.Copy(out, file); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	fileInfo, err := getImageInfo(filePath)
	if err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid image file"})
		return
	}

	sonarFile := models.SonarFile{
		ID:        fileID,
		Name:      header.Filename,
		Path:      filePath,
		Width:     fileInfo.Width,
		Height:    fileInfo.Height,
		Size:      header.Size,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := database.DB.Create(&sonarFile).Error; err != nil {
		os.Remove(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sonarFile)
}

func GetSonarImage(c *gin.Context) {
	id := c.Param("id")
	var file models.SonarFile
	if err := database.DB.First(&file, "id = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	if _, err := os.Stat(file.Path); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image file not found"})
		return
	}

	c.File(file.Path)
}

func getImageInfo(filePath string) (*FileInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	}

	return &FileInfo{
		Width:  config.Width,
		Height: config.Height,
	}, nil
}

func jsonResp(c *gin.Context, code int, data interface{}) {
	c.JSON(code, data)
}

func jsonErr(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

func mustJson(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
