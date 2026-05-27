package config

import "os"

type Config struct {
	ServerAddr   string
	Issuer       string
	DatabasePath string
	AdminSecret  string
}

func Load() *Config {
	return &Config{
		ServerAddr:   getEnv("SERVER_ADDR", ":8080"),
		Issuer:       getEnv("ISSUER", "http://localhost:8080"),
		DatabasePath: getEnv("DATABASE_PATH", "sso.db"),
		AdminSecret:  getEnv("ADMIN_SECRET", "change-me-in-production"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
