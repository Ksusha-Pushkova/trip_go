package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	LogLevel        string
	ShutdownTimeout time.Duration

	DatabaseURL             string
	DatabaseMaxConns        int32
	DatabaseMinConns        int32
	DatabaseMaxConnLifetime time.Duration
	DatabaseConnectTimeout  time.Duration
	DatabaseQueryTimeout    time.Duration
}

func Load() (Config, error) {
	dbURL, err := getRequiredEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := getDurationEnv("SHUTDOWN_TIMEOUT", 10*time.Second)
	if err != nil {
		return Config{}, err
	}

	maxConnLifetime, err := getDurationEnv("DATABASE_MAX_CONN_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}

	connectTimeout, err := getDurationEnv("DATABASE_CONNECT_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}

	queryTimeout, err := getDurationEnv("DATABASE_QUERY_TIMEOUT", 3*time.Second)
	if err != nil {
		return Config{}, err
	}

	maxConns, err := getInt32Env("DATABASE_MAX_CONNS", 10)
	if err != nil {
		return Config{}, err
	}

	minConns, err := getInt32Env("DATABASE_MIN_CONNS", 2)
	if err != nil {
		return Config{}, err
	}

	return Config{
		HTTPAddr:                getStringEnv("HTTP_ADDR", ":8080"),
		LogLevel:                getStringEnv("LOG_LEVEL", "info"),
		ShutdownTimeout:         shutdownTimeout,
		DatabaseURL:             dbURL,
		DatabaseMaxConns:        maxConns,
		DatabaseMinConns:        minConns,
		DatabaseMaxConnLifetime: maxConnLifetime,
		DatabaseConnectTimeout:  connectTimeout,
		DatabaseQueryTimeout:    queryTimeout,
	}, nil
}

func getRequiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}
	return value, nil
}

func getStringEnv(name, defaultVal string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return defaultVal
}

func getDurationEnv(name string, defaultVal time.Duration) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultVal, nil
	}

	d, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf("invalid duration in %s: %w", name, err)
	}
	return d, nil
}

func getInt32Env(name string, defaultVal int32) (int32, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultVal, nil
	}

	n, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid integer in %s: %w", name, err)
	}
	return int32(n), nil
}