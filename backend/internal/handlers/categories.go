package handlers

import (
	"net/http"
	"sonar-annotation-backend/internal/database"
	"sonar-annotation-backend/internal/models"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	HeaderUserID = "X-User-Id"
)

func GetUserID(c *gin.Context) string {
	uid := c.GetHeader(HeaderUserID)
	if uid != "" {
		return uid
	}
	uid = c.Query("userId")
	if uid != "" {
		return uid
	}
	return ""
}

func ListCategories(c *gin.Context) {
	userID := GetUserID(c)

	var globalCats []models.Category
	if err := database.DB.Where("user_id IS NULL").Order("is_builtin DESC, name ASC").Find(&globalCats).Error; err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	seedRequired := len(globalCats) == 0 && userID == ""
	if seedRequired {
		defaultCategories := []models.Category{
			{Name: "礁石", Color: "#ff4d4f", Description: "水下礁石", IsBuiltin: true},
			{Name: "沉船", Color: "#faad14", Description: "沉船残骸", IsBuiltin: true},
			{Name: "管线", Color: "#1890ff", Description: "海底管线", IsBuiltin: true},
			{Name: "锚", Color: "#722ed1", Description: "船锚", IsBuiltin: true},
			{Name: "渔网", Color: "#13c2c2", Description: "废弃渔网", IsBuiltin: true},
			{Name: "其他", Color: "#8c8c8c", Description: "其他目标", IsBuiltin: true},
		}
		for i := range defaultCategories {
			defaultCategories[i].ID = uuid.New().String()
			defaultCategories[i].CreatedAt = time.Now()
			defaultCategories[i].UpdatedAt = time.Now()
			database.DB.Create(&defaultCategories[i])
		}
		globalCats = defaultCategories
	}

	all := make([]models.Category, 0, len(globalCats))
	all = append(all, globalCats...)

	if userID != "" {
		var userCats []models.Category
		if err := database.DB.Where("user_id = ?", userID).Order("name ASC").Find(&userCats).Error; err == nil {
			all = append(all, userCats...)
		}
	}

	c.JSON(http.StatusOK, all)
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Color       string `json:"color" binding:"required,max=20"`
	Description string `json:"description" binding:"max=500"`
}

func CreateCategory(c *gin.Context) {
	userID := GetUserID(c)
	if userID == "" {
		jsonErr(c, http.StatusUnauthorized, "User ID is required")
		return
	}

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonErr(c, http.StatusBadRequest, err.Error())
		return
	}

	if !isValidHexColor(req.Color) {
		jsonErr(c, http.StatusBadRequest, "Invalid color format, expected #RRGGBB")
		return
	}

	var globalConflict models.Category
	if err := database.DB.Where("user_id IS NULL AND name = ?", req.Name).First(&globalConflict).Error; err == nil {
		jsonErr(c, http.StatusConflict, "Category name conflicts with a global template")
		return
	}

	var userConflict models.Category
	if err := database.DB.Where("user_id = ? AND name = ?", userID, req.Name).First(&userConflict).Error; err == nil {
		jsonErr(c, http.StatusConflict, "You already have a category with this name")
		return
	}

	category := models.Category{
		ID:          uuid.New().String(),
		UserID:      &userID,
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
		IsBuiltin:   false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := database.DB.Create(&category).Error; err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, category)
}

type UpdateCategoryRequest struct {
	Name        *string `json:"name"`
	Color       *string `json:"color"`
	Description *string `json:"description"`
}

func UpdateCategory(c *gin.Context) {
	userID := GetUserID(c)
	if userID == "" {
		jsonErr(c, http.StatusUnauthorized, "User ID is required")
		return
	}

	id := c.Param("id")

	var existing models.Category
	if err := database.DB.First(&existing, "id = ?", id).Error; err != nil {
		jsonErr(c, http.StatusNotFound, "Category not found")
		return
	}

	if existing.IsBuiltin || existing.UserID == nil {
		jsonErr(c, http.StatusForbidden, "Cannot modify built-in or global categories")
		return
	}

	if existing.UserID != nil && *existing.UserID != userID {
		jsonErr(c, http.StatusForbidden, "This category does not belong to you")
		return
	}

	var req UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonErr(c, http.StatusBadRequest, err.Error())
		return
	}

	if req.Name != nil && *req.Name != existing.Name {
		var nameConflict models.Category
		query := database.DB.Where("name = ? AND id != ?", *req.Name, id).
			Where("user_id IS NULL OR user_id = ?", userID)
		if err := query.First(&nameConflict).Error; err == nil {
			jsonErr(c, http.StatusConflict, "Category name already in use")
			return
		}
		existing.Name = *req.Name
	}

	if req.Color != nil {
		if !isValidHexColor(*req.Color) {
			jsonErr(c, http.StatusBadRequest, "Invalid color format")
			return
		}
		existing.Color = *req.Color
	}

	if req.Description != nil {
		existing.Description = *req.Description
	}

	existing.UpdatedAt = time.Now()

	if err := database.DB.Save(&existing).Error; err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, existing)
}

func DeleteCategory(c *gin.Context) {
	userID := GetUserID(c)
	if userID == "" {
		jsonErr(c, http.StatusUnauthorized, "User ID is required")
		return
	}

	id := c.Param("id")

	var existing models.Category
	if err := database.DB.First(&existing, "id = ?", id).Error; err != nil {
		jsonErr(c, http.StatusNotFound, "Category not found")
		return
	}

	if existing.IsBuiltin {
		jsonErr(c, http.StatusForbidden, "Cannot delete built-in categories")
		return
	}

	if existing.UserID == nil {
		jsonErr(c, http.StatusForbidden, "Cannot delete global categories")
		return
	}

	if existing.UserID != nil && *existing.UserID != userID {
		jsonErr(c, http.StatusForbidden, "This category does not belong to you")
		return
	}

	tx := database.DB.Begin()
	if tx.Error != nil {
		jsonErr(c, http.StatusInternalServerError, tx.Error.Error())
		return
	}

	if err := tx.Model(&models.Annotation{}).Where("category_id = ?", id).
		Update("category_id", nil).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Delete(&existing).Error; err != nil {
		tx.Rollback()
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	if err := tx.Commit().Error; err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Category deleted successfully"})
}

func isValidHexColor(s string) bool {
	if len(s) != 7 && len(s) != 4 {
		return false
	}
	if s[0] != '#' {
		return false
	}
	for i := 1; i < len(s); i++ {
		ch := s[i]
		valid := (ch >= '0' && ch <= '9') ||
			(ch >= 'a' && ch <= 'f') ||
			(ch >= 'A' && ch <= 'F')
		if !valid {
			return false
		}
	}
	return true
}
