package config

import "os"

type Config struct {
	Addr           string
	GitHubToken    string
	DeepSeekAPIKey string
}

func Load() Config {
	return Config{
		Addr:           envOr("ADDR", ":8080"),
		GitHubToken:    os.Getenv("GITHUB_TOKEN"),
		DeepSeekAPIKey: os.Getenv("DEEPSEEK_API_KEY"),
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
