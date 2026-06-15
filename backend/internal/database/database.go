package database

import (
	"context"
	"encoding/json"
	"fmt"
	"sonar-annotation-backend/internal/config"
	"sonar-annotation-backend/internal/models"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type DBType = *gorm.DB

var (
	DB    *gorm.DB
	Redis *redis.Client
	Ctx   = context.Background()
)

func InitPostgreSQL(cfg *config.Config) error {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.PostgresHost, cfg.PostgresPort, cfg.PostgresUser,
		cfg.PostgresPass, cfg.PostgresDB, cfg.PostgresSSL,
	)

	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	return DB.AutoMigrate(
		&models.SonarFile{},
		&models.Annotation{},
		&models.Category{},
		&models.Snapshot{},
	)
}

func InitRedis(cfg *config.Config) error {
	db, _ := strconv.Atoi(strconv.Itoa(cfg.RedisDB))
	Redis = redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%s", cfg.RedisHost, cfg.RedisPort),
		Password: cfg.RedisPass,
		DB:       db,
	})

	return Redis.Ping(Ctx).Err()
}

func GetOnlineUsers(fileID string) (map[string]string, error) {
	key := fmt.Sprintf("sonar:online:%s", fileID)
	users, err := Redis.HGetAll(Ctx, key).Result()
	return users, err
}

func AddOnlineUser(fileID, userID, userName string) error {
	key := fmt.Sprintf("sonar:online:%s", fileID)
	return Redis.HSet(Ctx, key, userID, userName).Err()
}

func RemoveOnlineUser(fileID, userID string) error {
	key := fmt.Sprintf("sonar:online:%s", fileID)
	return Redis.HDel(Ctx, key, userID).Err()
}

func CacheAnnotations(fileID string, annotations []models.Annotation) error {
	key := fmt.Sprintf("sonar:annotations:%s", fileID)
	data, _ := jsonMarshal(annotations)
	return Redis.Set(Ctx, key, data, 5*time.Minute).Err()
}

func GetCachedAnnotations(fileID string) ([]models.Annotation, error) {
	key := fmt.Sprintf("sonar:annotations:%s", fileID)
	data, err := Redis.Get(Ctx, key).Result()
	if err != nil {
		return nil, err
	}
	var annotations []models.Annotation
	err = jsonUnmarshal([]byte(data), &annotations)
	return annotations, err
}

func jsonMarshal(v interface{}) (string, error) {
	b, err := jsonMarshalBytes(v)
	return string(b), err
}

func jsonMarshalBytes(v interface{}) ([]byte, error) {
	type jsonMarshaler interface {
		MarshalJSON() ([]byte, error)
	}
	if m, ok := v.(jsonMarshaler); ok {
		return m.MarshalJSON()
	}
	return jsonMarshalFallback(v)
}

func jsonUnmarshal(data []byte, v interface{}) error {
	type jsonUnmarshaler interface {
		UnmarshalJSON([]byte) error
	}
	if m, ok := v.(jsonUnmarshaler); ok {
		return m.UnmarshalJSON(data)
	}
	return jsonUnmarshalFallback(data, v)
}

func jsonMarshalFallback(v interface{}) ([]byte, error) {
	switch val := v.(type) {
	case []models.Annotation:
		buf := []byte{'['}
		for i, a := range val {
			if i > 0 {
				buf = append(buf, ',')
			}
			pointsJSON, _ := a.Points.MarshalJSON()
			buf = append(buf, fmt.Sprintf(
				`{"id":"%s","fileId":"%s","type":"%s","points":%s,"categoryId":"%s","label":"%s","color":"%s","createdBy":"%s","createdAt":"%s","updatedAt":"%s"}`,
				a.ID, a.FileID, a.Type, string(pointsJSON), a.CategoryID, a.Label, a.Color, a.CreatedBy,
				a.CreatedAt.Format(time.RFC3339), a.UpdatedAt.Format(time.RFC3339),
			)...)
		}
		buf = append(buf, ']')
		return buf, nil
	}
	return nil, nil
}

func jsonUnmarshalFallback(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
