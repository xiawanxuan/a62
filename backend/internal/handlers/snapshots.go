package handlers

import (
	"encoding/json"
	"net/http"
	"sonar-annotation-backend/internal/config"
	"sonar-annotation-backend/internal/database"
	"sonar-annotation-backend/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func ListSnapshots(c *gin.Context) {
	fileID := c.Param("fileId")

	var snapshots []models.Snapshot
	if err := database.DB.Where("file_id = ?", fileID).
		Order("created_at DESC").
		Limit(30).
		Find(&snapshots).Error; err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, snapshots)
}

func RestoreSnapshot(c *gin.Context) {
	id := c.Param("id")

	var snapshot models.Snapshot
	if err := database.DB.First(&snapshot, "id = ?", id).Error; err != nil {
		jsonErr(c, http.StatusNotFound, "Snapshot not found")
		return
	}

	var annotations []models.Annotation
	if err := json.Unmarshal(snapshot.Annotations, &annotations); err != nil {
		jsonErr(c, http.StatusInternalServerError, "Invalid snapshot data")
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		jsonErr(c, http.StatusInternalServerError, tx.Error.Error())
		return
	}

	if err := tx.Where("file_id = ?", snapshot.FileID).Delete(&models.Annotation{}).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	for i := range annotations {
		annotations[i].ID = ""
		annotations[i].CreatedAt = time.Now()
		annotations[i].UpdatedAt = time.Now()
	}

	if len(annotations) > 0 {
		if err := tx.Create(&annotations).Error; err != nil {
			tx.Rollback()
			jsonErr(c, http.StatusInternalServerError, err.Error())
			return
		}
	}

	if err := tx.Model(&models.SonarFile{}).Where("id = ?", snapshot.FileID).
		Update("annotation_count", len(annotations)).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	newSnapshot := models.Snapshot{
		FileID:      snapshot.FileID,
		Annotations: snapshot.Annotations,
		CreatedBy:   snapshot.CreatedBy,
		Message:     "Restore from snapshot",
		CreatedAt:   time.Now(),
	}
	if err := tx.Create(&newSnapshot).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	tx.Commit()

	_ = database.CacheAnnotations(snapshot.FileID, []models.Annotation{})

	c.JSON(http.StatusOK, gin.H{
		"message":     "Restored successfully",
		"annotations": annotations,
	})
}

func CreateSnapshot(fileID, createdBy, message string) error {
	cfg := config.Load()

	var annotations []models.Annotation
	if err := database.DB.Where("file_id = ?", fileID).Find(&annotations).Error; err != nil {
		return err
	}

	annotationsJSON, err := json.Marshal(annotations)
	if err != nil {
		return err
	}

	var maxVersion int
	database.DB.Model(&models.Snapshot{}).Where("file_id = ?", fileID).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)

	snapshot := models.Snapshot{
		FileID:      fileID,
		Annotations: datatypes.JSON(annotationsJSON),
		CreatedBy:   createdBy,
		Message:     message,
		Version:     maxVersion + 1,
		CreatedAt:   time.Now(),
	}

	if err := database.DB.Create(&snapshot).Error; err != nil {
		return err
	}

	var count int64
	database.DB.Model(&models.Snapshot{}).Where("file_id = ?", fileID).Count(&count)
	if count > int64(cfg.MaxSnapshots) {
		var oldest models.Snapshot
		database.DB.Where("file_id = ?", fileID).Order("created_at ASC").First(&oldest)
		database.DB.Delete(&oldest)
	}

	return nil
}

func createSnapshotTx(tx *gorm.DB, fileID, createdBy, message string) error {
	cfg := config.Load()

	var annotations []models.Annotation
	if err := tx.Where("file_id = ?", fileID).Find(&annotations).Error; err != nil {
		return err
	}

	annotationsJSON, err := json.Marshal(annotations)
	if err != nil {
		return err
	}

	var maxVersion int
	tx.Model(&models.Snapshot{}).Where("file_id = ?", fileID).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion)

	snapshot := models.Snapshot{
		FileID:      fileID,
		Annotations: datatypes.JSON(annotationsJSON),
		CreatedBy:   createdBy,
		Message:     message,
		Version:     maxVersion + 1,
		CreatedAt:   time.Now(),
	}

	if err := tx.Create(&snapshot).Error; err != nil {
		return err
	}

	var count int64
	tx.Model(&models.Snapshot{}).Where("file_id = ?", fileID).Count(&count)
	if count > int64(cfg.MaxSnapshots) {
		var oldest models.Snapshot
		tx.Where("file_id = ?", fileID).Order("created_at ASC").First(&oldest)
		tx.Delete(&oldest)
	}

	return nil
}
