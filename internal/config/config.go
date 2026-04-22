package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPAddr       string
	DatabaseURL    string
	JWTSecret      []byte
	JWTExpiration  time.Duration
	BcryptCost     int
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	addr := getEnv("HTTP_ADDR", ":8080")

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = buildDSNFromParts()
	}
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	expHours, err := strconv.Atoi(getEnv("JWT_EXPIRATION_HOURS", "24"))
	if err != nil {
		return nil, fmt.Errorf("invalid JWT_EXPIRATION_HOURS: %w", err)
	}

	bcryptCost, err := strconv.Atoi(getEnv("BCRYPT_COST", "12"))
	if err != nil {
		return nil, fmt.Errorf("invalid BCRYPT_COST: %w", err)
	}

	return &Config{
		HTTPAddr:      addr,
		DatabaseURL:   dbURL,
		JWTSecret:     []byte(secret),
		JWTExpiration: time.Duration(expHours) * time.Hour,
		BcryptCost:    bcryptCost,
	}, nil
}

func buildDSNFromParts() string {
	host := os.Getenv("DB_HOST")
	if host == "" {
		return ""
	}
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	pass := os.Getenv("DB_PASSWORD")
	name := getEnv("DB_NAME", "postgres")
	sslMode := getEnv("DB_SSLMODE", "disable")
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, pass, name, sslMode)
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
