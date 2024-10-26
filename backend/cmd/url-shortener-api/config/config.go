package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port               string
	SafeBrowsingAPIKey string
	DBConnectionString string
}

func LoadConfig() Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, relying on environment variables")
	}

	config := Config{
		Port:               getEnv("PORT", "8080"),
		SafeBrowsingAPIKey: getEnv("SAFE_BROWSING_API_KEY", ""),
		DBConnectionString: getEnv("DB_CONNECTION_STRING", ""),
	}

	return config
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
