package handlers

import (
	"net/http"
	"sonar-annotation-backend/internal/database"
	"sonar-annotation-backend/internal/models"
	"time"

	"github.com/gin-gonic/gin"
)

func ListCategories(c *gin.Context) {
	var categories []models.Category
	if err := database.DB.Order("name ASC").Find(&categories).Error; err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	if len(categories) == 0 {
		defaultCategories := []models.Category{
			{Name: "礁石", Color: "#ff4d4f", Description: "水下礁石"},
			{Name: "沉船", Color: "#faad14", Description: "沉船残骸"},
			{Name: "管线", Color: "#1890ff", Description: "海底管线"},
			{Name: "锚", Color: "#722ed1", Description: "船锚"},
			{Name: "渔网", Color: "#13c2c2", Description: "废弃渔网"},
			{Name: "其他", Color: "#8c8c8c", Description: "其他目标"},
		}
		for i := range defaultCategories {
			defaultCategories[i].CreatedAt = time.Now()
			database.DB.Create(&defaultCategories[i])
		}
		categories = defaultCategories
	}

	c.JSON(http.StatusOK, categories)
}

type CreateCategoryRequest struct {
	Name        string `json:"name" binding:"required,max=100"`
	Color       string `json:"color" binding:"required,max=20"`
	Description string `json:"description"`
}

func CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		jsonErr(c, http.StatusBadRequest, err.Error())
		return
	}

	var existing models.Category
	if err := database.DB.Where("name = ?", req.Name).First(&existing).Error; err == nil {
		jsonErr(c, http.StatusConflict, "Category already exists")
		return
	}

	category := models.Category{
		Name:        req.Name,
		Color:       req.Color,
		Description: req.Description,
		CreatedAt:   time.Now(),
	}

	if err := database.DB.Create(&category).Error; err != nil {
		jsonErr(c, http.StatusInternalServerError, err.Error())
		return
	}

	c.JSON(http.StatusOK, category)
}
