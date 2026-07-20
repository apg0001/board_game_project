package config

import (
	"os"
	"strings"
)

type Config struct {
	HTTPAddr       string
	AllowedOrigins map[string]struct{}
	DatabaseURL    string
	RedisURL       string
}

func Load() Config {
	return Config{
		HTTPAddr:       env("HTTP_ADDR", ":4000"),
		AllowedOrigins: parseOrigins(env("CORS_ORIGINS", "http://localhost:5173,http://127.0.0.1:5173")),
		DatabaseURL:    env("DATABASE_URL", ""),
		RedisURL:       env("REDIS_URL", ""),
	}
}

func env(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func parseOrigins(raw string) map[string]struct{} {
	origins := make(map[string]struct{})
	for _, origin := range strings.Split(raw, ",") {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			origins[origin] = struct{}{}
		}
	}
	return origins
}
