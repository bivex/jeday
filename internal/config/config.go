package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
)

// Config stores all configuration of the application.
// The values are read by godotenv from a .env file or environment variables.
type Config struct {
	DBSource            string
	ServerAddress       string
	TokenSymmetricKey   string
	AccessTokenDuration time.Duration
}

// Load reads configuration from file or environment variables.
func Load(path string) (config Config, err error) {
	godotenv.Load(path + "/.env") // Load .env file if it exists, otherwise use env variables

	config.DBSource = os.Getenv("DB_URL")
	if config.DBSource == "" {
		config.DBSource = "postgresql://user:secret@localhost:5432/auth_db?sslmode=disable"
	}

	config.ServerAddress = os.Getenv("SERVER_ADDRESS")
	if config.ServerAddress == "" {
		config.ServerAddress = "0.0.0.0:8080"
	}

	config.TokenSymmetricKey = os.Getenv("TOKEN_SYMMETRIC_KEY")
	if config.TokenSymmetricKey == "" {
		config.TokenSymmetricKey = "12345678901234567890123456789012"
	}

	durStr := os.Getenv("ACCESS_TOKEN_DURATION")
	if durStr == "" {
		durStr = "15m"
	}
	config.AccessTokenDuration, _ = time.ParseDuration(durStr)

	return
}
