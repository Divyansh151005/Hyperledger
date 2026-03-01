package config

import (
	"log"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

// Config holds all configuration for the middleware
type Config struct {
	// API Server
	APIHost    string
	APIPort    int
	LogLevel   logrus.Level

	// Database
	DatabaseURL string

	// Object Storage (S3)
	S3Config S3Config

	// Fabric Network
	FabricConfig FabricConfig

	// Encryption
	EncryptionConfig EncryptionConfig
}

// S3Config holds S3 configuration
type S3Config struct {
	Endpoint  string
	Bucket    string
	AccessKey string
	SecretKey string
	Region    string
}

// FabricConfig holds Fabric network configuration
type FabricConfig struct {
	NetworkConfigPath string
	ChannelName      string
	ChaincodeNames   ChaincodeNames
}

// ChaincodeNames holds chaincode names
type ChaincodeNames struct {
	MedicalRecords   string
	Authorization   string
	AnonymousSharing string
}

// EncryptionConfig holds encryption configuration
type EncryptionConfig struct {
	KeyStorePath    string
	Algorithm       string
	KeySize         int
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		APIHost:    getEnv("API_HOST", "0.0.0.0"),
		APIPort:    getEnvAsInt("API_PORT", 8080),
		LogLevel:   getLogLevel(getEnv("API_LOG_LEVEL", "info")),
		DatabaseURL: getEnv("DATABASE_URL", "postgresql://user:password@localhost:5432/medical_middleware?sslmode=disable"),
		S3Config: S3Config{
			Endpoint:  getEnv("S3_ENDPOINT", ""),
			Bucket:    getEnv("S3_BUCKET", "medical-records"),
			AccessKey: getEnv("S3_ACCESS_KEY", ""),
			SecretKey: getEnv("S3_SECRET_KEY", ""),
			Region:    getEnv("S3_REGION", "us-east-1"),
		},
		FabricConfig: FabricConfig{
			NetworkConfigPath: getEnv("FABRIC_NETWORK_CONFIG_PATH", "./network/connection-profile.json"),
			ChannelName:       getEnv("FABRIC_CHANNEL_NAME", "medical-main-channel"),
			ChaincodeNames: ChaincodeNames{
				MedicalRecords:   getEnv("FABRIC_CHAINCODE_MEDICAL_RECORDS", "medical-records"),
				Authorization:    getEnv("FABRIC_CHAINCODE_AUTHORIZATION", "authorization-consent"),
				AnonymousSharing: getEnv("FABRIC_CHAINCODE_ANONYMOUS_SHARING", "anonymous-sharing"),
			},
		},
		EncryptionConfig: EncryptionConfig{
			KeyStorePath: getEnv("ENCRYPTION_KEY_STORE_PATH", "./keystore"),
			Algorithm:    getEnv("ENCRYPTION_ALGORITHM", "AES-256-GCM"),
			KeySize:      getEnvAsInt("ENCRYPTION_KEY_SIZE", 32), // 32 bytes = 256 bits
		},
	}

	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := getEnv(key, "")
	if valueStr == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(valueStr)
	if err != nil {
		log.Printf("Invalid integer value for %s: %s, using default: %d", key, valueStr, defaultValue)
		return defaultValue
	}
	return value
}

func getLogLevel(level string) logrus.Level {
	switch level {
	case "debug":
		return logrus.DebugLevel
	case "info":
		return logrus.InfoLevel
	case "warn":
		return logrus.WarnLevel
	case "error":
		return logrus.ErrorLevel
	default:
		return logrus.InfoLevel
	}
}
