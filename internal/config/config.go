package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
    MinioEndpoint  string
    MinioAccessKey string
    MinioSecretKey string
    MinioSSL       bool
    MongoURI       string
    MongoDatabase  string
    ServerPort     string
}

func LoadConfig() *Config {
    if err := godotenv.Load(); err != nil {
        log.Println("No .env file found")
    }
    
    return &Config{
        MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost:9000"),
        MinioAccessKey: getEnv("MINIO_ACCESS_KEY", "minioadmin"),
        MinioSecretKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
        MinioSSL:       getEnvAsBool("MINIO_SSL", false),
        MongoURI:       getEnv("MONGO_URI", "mongodb://localhost:27017"),
        MongoDatabase:  getEnv("MONGO_DATABASE", "file_storage"),
        ServerPort:     getEnv("SERVER_PORT", ":8080"),
    }
}

func getEnv(key, defaultValue string) string {
    if value, exists := os.LookupEnv(key); exists {
        return value
    }
    return defaultValue
}

func getEnvAsBool(key string, defaultValue bool) bool {
    if value, exists := os.LookupEnv(key); exists {
        boolValue, err := strconv.ParseBool(value)
        if err != nil {
            return defaultValue
        }
        return boolValue
    }
    return defaultValue
}