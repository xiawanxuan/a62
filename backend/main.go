package main

import (
	"log"
	"sonar-annotation-backend/internal/config"
	"sonar-annotation-backend/internal/database"
	"sonar-annotation-backend/internal/handlers"
	"sonar-annotation-backend/internal/ws"

	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()

	if err := database.InitPostgreSQL(cfg); err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}

	if err := database.InitRedis(cfg); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}

	hub := ws.NewHub()
	go hub.Run()

	r := gin.Default()

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	api := r.Group("/api")
	{
		files := api.Group("/files")
		{
			files.GET("", handlers.ListSonarFiles)
			files.POST("", handlers.UploadSonarFile)
			files.GET("/:id", handlers.GetSonarFile)
			files.GET("/:id/image", handlers.GetSonarImage)
		}

		annotations := api.Group("/annotations")
		{
			annotations.GET("/file/:fileId", handlers.ListAnnotations)
			annotations.POST("", handlers.CreateAnnotation)
			annotations.PUT("/:id", handlers.UpdateAnnotation)
			annotations.DELETE("/:id", handlers.DeleteAnnotation)
		}

		snapshots := api.Group("/snapshots")
		{
			snapshots.GET("/file/:fileId", handlers.ListSnapshots)
			snapshots.POST("/restore/:id", handlers.RestoreSnapshot)
		}

		categories := api.Group("/categories")
		{
			categories.GET("", handlers.ListCategories)
			categories.POST("", handlers.CreateCategory)
		}
	}

	r.GET("/ws/annotate/:fileId", func(c *gin.Context) {
		ws.ServeWebSocket(hub, c)
	})

	log.Printf("Server starting on port %s", cfg.ServerPort)
	if err := r.Run(":" + cfg.ServerPort); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
