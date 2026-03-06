package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config stores all configuration of the application.
// The values are read by godotenv from a .env file or environment variables.
type Config struct {
	DBSource              string
	ServerAddress         string
	ServerPrefork         bool
	TokenSymmetricKey     string
	AccessTokenDuration   time.Duration
	RegistrationBatchSize int
	RegistrationBatchWait time.Duration
	WorkerInterval        time.Duration
	WorkerUpgradeLimit    int32
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

	config.ServerPrefork = parseBoolEnv("SERVER_PREFORK", false)

	config.TokenSymmetricKey = os.Getenv("TOKEN_SYMMETRIC_KEY")
	if config.TokenSymmetricKey == "" {
		config.TokenSymmetricKey = "12345678901234567890123456789012"
	}

	durStr := os.Getenv("ACCESS_TOKEN_DURATION")
	if durStr == "" {
		durStr = "15m"
	}
	config.AccessTokenDuration, _ = time.ParseDuration(durStr)

	config.RegistrationBatchSize = parseIntEnv("REGISTRATION_BATCH_SIZE", 100)
	config.RegistrationBatchWait = parseDurationEnv("REGISTRATION_BATCH_WAIT", 10*time.Millisecond)
	config.WorkerInterval = parseDurationEnv("WORKER_INTERVAL", 2*time.Second)
	config.WorkerUpgradeLimit = int32(parseIntEnv("WORKER_UPGRADE_LIMIT", 10))

	return
}

func parseBoolEnv(key string, fallback bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func parseDurationEnv(key string, fallback time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}

	return parsed
}

func parseIntEnv(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}

	return parsed
}
