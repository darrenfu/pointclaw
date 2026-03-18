package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	RedisAddr   string
	DatabaseURL string
	PoolSize    int
	MaxRetries  int
	BudgetDaily int // max scrapes per day
}

func Load() (*Config, error) {
	poolSize, _ := strconv.Atoi(getEnv("POOL_SIZE", "3"))
	maxRetries, _ := strconv.Atoi(getEnv("MAX_RETRIES", "5"))
	budgetDaily, _ := strconv.Atoi(getEnv("BUDGET_DAILY", "500"))

	cfg := &Config{
		RedisAddr:   getEnv("REDIS_ADDR", "localhost:6379"),
		DatabaseURL: getEnv("DATABASE_URL", ""),
		PoolSize:    poolSize,
		MaxRetries:  maxRetries,
		BudgetDaily: budgetDaily,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
