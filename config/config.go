package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort         string
	DBUser          string
	DBPassword      string
	DBHost          string
	DBPort          string
	DBName          string
	JWTSecret       string
	EncKeyHex       string
	AllowOrigin     string
	// AnthropicAPIKey: opsional. Kalau diisi, generate jadwal habit
	// (handlers/habit_handler.go) memakai Claude API supaya jadwalnya
	// lebih natural & kontekstual. Kalau kosong, otomatis fallback ke
	// generator statis berbasis aturan (tetap jalan, cuma kurang lentur).
	AnthropicAPIKey string
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
		DBPassword:  getEnv("DB_PASSWORD", "bBopnMbtAKwIRakAKwBSxnAcUHmaWoTY"),
		DBHost:      getEnv("DB_HOST", "mysql.railway.internal"),
		DBPort:      getEnv("DB_PORT", "3306"),
		DBName:      getEnv("DB_NAME", "railway"),
		JWTSecret:   getEnv("JWT_SECRET", "change-this-jwt-secret"),
		EncKeyHex:   getEnv("APP_ENC_KEY", "39a33c66bece4d9450357bdba3341965b40024d9105da0ddc63b6ea0279b6827"),
		AllowOrigin: getEnv("ALLOW_ORIGIN", "*"),
		AnthropicAPIKey: getEnv("ANTHROPIC_API_KEY", ""),
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