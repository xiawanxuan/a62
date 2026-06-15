package handlers

import (
	"encoding/json"
	"net/http"
	"sonar-annotation-backend/internal/database"
	"sonar-annotation-backend/internal/models"
	"sonar-annotation-backend/pkg/validation"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func ListAnnotations(c *gin.Context) {
	fileID := c.Param("fileId")

	if cached, err := database.GetCachedAnnotations(fileID); err == nil && len(cached) > 0 {
		c.JSON(http.StatusOK, cached)
		return
	}

	var annotations []models.Annotation
	if err := database.DB.Where("file_id = ?", fileID).Order("created_at ASC").Find(&annotations).Error; err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	_ = database.CacheAnnotations(fileID, annotations)
	c.JSON(http.StatusOK, annotations)
}

type CreateAnnotationRequest struct {
	FileID     string            `json:"fileId" binding:"required"`
	Type       string            `json:"type" binding:"required,oneof=rectangle polygon"`
	Points     []models.Point    `json:"points" binding:"required,min=2"`
	CategoryID string            `json:"categoryId" binding:"required"`
	Label      string            `json:"label"`
	Color      string            `json:"color"`
	CreatedBy  string            `json:"createdBy" binding:"required"`
	Confidence *float64          `json:"confidence,omitempty"`
}

func CreateAnnotation(c *gin.Context) {
	var req CreateAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonErr(c, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if !validateAnnotation(req.Type, req.Points) {
		jsonErr(c, http.StatusBadRequest, "Invalid annotation points")
		return
	}

	pointsJSON, _ := json.Marshal(req.Points)

	annotation := models.Annotation{
		ID:         uuid.New().String(),
		FileID:     req.FileID,
		Type:       req.Type,
		Points:     datatypes.JSON(pointsJSON),
		CategoryID: req.CategoryID,
		Label:      req.Label,
		Color:      req.Color,
		CreatedBy:  req.CreatedBy,
		Confidence: req.Confidence,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		jsonErr(c, http.StatusInternalServerError, tx.Error.Error())
		return
	}

	if err := tx.Create(&annotation).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Model(&models.SonarFile{}).Where("id = ?", req.FileID).
		UpdateColumn("annotation_count", gormExpr("annotation_count + 1")).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := createSnapshotTx(tx, req.FileID, req.CreatedBy, "Create annotation: "+req.Label); err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	tx.Commit()

	_ = database.CacheAnnotations(req.FileID, []models.Annotation{})

	c.JSON(http.StatusOK, annotation)
}

type UpdateAnnotationRequest struct {
	Points     []models.Point `json:"points"`
	CategoryID string         `json:"categoryId"`
	Label      string         `json:"label"`
	Color      string         `json:"color"`
}

func UpdateAnnotation(c *gin.Context) {
	id := c.Param("id")

	var existing models.Annotation
	if err := database.DB.First(&existing, "id = ?", id).Error; err != nil {
		jsonErr(c, http.StatusNotFound, "Annotation not found")
		return
	}

	var req UpdateAnnotationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonErr(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Points != nil {
		if !validateAnnotation(existing.Type, req.Points) {
			jsonErr(c, http.StatusBadRequest, "Invalid annotation points")
			return
		}
		pointsJSON, _ := json.Marshal(req.Points)
		existing.Points = datatypes.JSON(pointsJSON)
	}

	if req.CategoryID != "" {
		existing.CategoryID = req.CategoryID
	}
	if req.Label != "" {
		existing.Label = req.Label
	}
	if req.Color != "" {
		existing.Color = req.Color
	}
	existing.UpdatedAt = time.Now()

	tx := database.DB.Begin()
	if tx.Error != nil {
		jsonErr(c, http.StatusInternalServerError, tx.Error.Error())
		return
	}

	if err := tx.Save(&existing).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := createSnapshotTx(tx, existing.FileID, existing.CreatedBy, "Update annotation"); err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	tx.Commit()

	_ = database.CacheAnnotations(existing.FileID, []models.Annotation{})

	c.JSON(http.StatusOK, existing)
}

func DeleteAnnotation(c *gin.Context) {
	id := c.Param("id")

	var existing models.Annotation
	if err := database.DB.First(&existing, "id = ?", id).Error; err != nil {
		jsonErr(c, http.StatusNotFound, "Annotation not found")
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		jsonErr(c, http.StatusInternalServerError, tx.Error.Error())
		return
	}

	if err := tx.Delete(&existing).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Model(&models.SonarFile{}).Where("id = ?", existing.FileID).
		UpdateColumn("annotation_count", gormExpr("annotation_count - 1")).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := createSnapshotTx(tx, existing.FileID, existing.CreatedBy, "Delete annotation"); err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	tx.Commit()

	_ = database.CacheAnnotations(existing.FileID, []models.Annotation{})

	c.JSON(http.StatusOK, gin.H{"message": "Deleted successfully"})
}

func validateAnnotation(typ string, points []models.Point) bool {
	errs := validation.ValidateAnnotation(typ, points)
	return len(errs) == 0
}

func gormExpr(expr string) interface{} {
	return database.DB.Raw(expr)
}
