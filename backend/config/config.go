package config

import "os"

// Config holds all application configuration loaded from environment variables.
type Config struct {
	ListenAddr  string
	S3Endpoint  string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool
	S3Region    string
}

// Load returns a Config populated from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		ListenAddr:  getEnv("LISTEN_ADDR", ":8080"),
		S3Endpoint:  getEnv("S3_ENDPOINT", "localhost:9000"),
		S3AccessKey: getEnv("S3_ACCESS_KEY", "minioadmin"),
		S3SecretKey: getEnv("S3_SECRET_KEY", "minioadmin"),
		S3UseSSL:    getEnv("S3_USE_SSL", "false") == "true",
		S3Region:    getEnv("S3_REGION", "us-east-1"),
	}
}

func getEnv(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
