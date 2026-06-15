package config

import (
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type Config struct {
	ServerPort     string
	PostgresHost   string
	PostgresPort   string
	PostgresUser   string
	PostgresPass   string
	PostgresDB     string
	PostgresSSL    string
	RedisHost      string
	RedisPort      string
	RedisPass      string
	RedisDB        int
	UploadDir      string
	MaxUploadSize  int64
	WsSyncInterval int
	MaxSnapshots   int
}

func Load() *Config {
	envPath := filepath.Join("..", "configs", "backend.env")
	_ = godotenv.Load(envPath)
	_ = godotenv.Load()

	return &Config{
		ServerPort:     getEnv("SERVER_PORT", "8080"),
		PostgresHost:   getEnv("POSTGRES_HOST", "localhost"),
		PostgresPort:   getEnv("POSTGRES_PORT", "5432"),
		PostgresUser:   getEnv("POSTGRES_USER", "postgres"),
		PostgresPass:   getEnv("POSTGRES_PASSWORD", "postgres"),
		PostgresDB:     getEnv("POSTGRES_DB", "sonar_annotation"),
		PostgresSSL:    getEnv("POSTGRES_SSLMODE", "disable"),
		RedisHost:      getEnv("REDIS_HOST", "localhost"),
		RedisPort:      getEnv("REDIS_PORT", "6379"),
		RedisPass:      getEnv("REDIS_PASSWORD", ""),
		RedisDB:        getEnvInt("REDIS_DB", 0),
		UploadDir:      getEnv("UPLOAD_DIR", "./uploads"),
		MaxUploadSize:  getEnvInt64("MAX_UPLOAD_SIZE", 100*1024*1024),
		WsSyncInterval: getEnvInt("WS_SYNC_INTERVAL", 100),
		MaxSnapshots:   getEnvInt("MAX_SNAPSHOTS", 30),
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := parseInt(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func getEnvInt64(key string, defaultValue int64) int64 {
	if value, exists := os.LookupEnv(key); exists {
		if intVal, err := parseInt64(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

func parseInt(s string) (int, error) {
	var result int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int(c-'0')
	}
	return result, nil
}

func parseInt64(s string) (int64, error) {
	var result int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, nil
		}
		result = result*10 + int64(c-'0')
	}
	return result, nil
}
