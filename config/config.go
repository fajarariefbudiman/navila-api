package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort     string
	DBUser      string
	DBPassword  string
	DBHost      string
	DBPort      string
	DBName      string
	JWTSecret   string
	EncKeyHex   string // 64 hex chars => 32 bytes (AES-256)
	AllowOrigin string
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func Load() *Config {
		if err := godotenv.Load(); err != nil {
		log.Println("info: file .env tidak ditemukan, membaca dari environment variable sistem")
	}
	cfg := &Config{
		AppPort:     getEnv("APP_PORT", "8080"),
		DBUser:      getEnv("DB_USER", "root"),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBHost:      getEnv("DB_HOST", "127.0.0.1"),
		DBPort:      getEnv("DB_PORT", "3306"),
		DBName:      getEnv("DB_NAME", "goals_db"),
		JWTSecret:   getEnv("JWT_SECRET", "change-this-jwt-secret"),
		EncKeyHex:   getEnv("APP_ENC_KEY", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcd"),
		AllowOrigin: getEnv("ALLOW_ORIGIN", "*"),
	}
	if cfg.JWTSecret == "change-this-in-production" {
		log.Fatal("JWT_SECRET wajib diset di environment production")
	}

	return cfg

}
func (c *Config) MySQLDSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.DBUser, c.DBPassword, c.DBHost, c.DBPort, c.DBName,
	)
}