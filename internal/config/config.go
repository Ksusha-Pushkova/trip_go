package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTP            HTTPConfig
	Database        DatabaseConfig
	LogLevel        string
	ShutdownTimeout time.Duration
}

type HTTPConfig struct {
	Addr              string
	ReadTimeout       time.Duration
	ReadHeaderTimeout time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration
}

type DatabaseConfig struct {
	URL             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
	ConnectTimeout  time.Duration
	QueryTimeout    time.Duration
}

func Load() (Config, error) {
	dbURL, err := getRequiredEnv("DATABASE_URL")
	if err != nil {
		return Config{}, err
	}

	shutdownTimeout, err := getPositiveDurationEnv(
		"SHUTDOWN_TIMEOUT",
		10*time.Second,
	)
	if err != nil {
		return Config{}, err
	}

	readTimeout, err := getPositiveDurationEnv(
		"HTTP_READ_TIMEOUT",
		10*time.Second,
	)
	if err != nil {
		return Config{}, err
	}

	readHeaderTimeout, err := getPositiveDurationEnv(
		"HTTP_READ_HEADER_TIMEOUT",
		5*time.Second,
	)
	if err != nil {
		return Config{}, err
	}

	writeTimeout, err := getPositiveDurationEnv(
		"HTTP_WRITE_TIMEOUT",
		15*time.Second,
	)
	if err != nil {
		return Config{}, err
	}

	idleTimeout, err := getPositiveDurationEnv(
		"HTTP_IDLE_TIMEOUT",
		60*time.Second,
	)
	if err != nil {
		return Config{}, err
	}

	maxConnLifetime, err := getPositiveDurationEnv(
		"DATABASE_MAX_CONN_LIFETIME",
		30*time.Minute,
	)
	if err != nil {
		return Config{}, err
	}

	maxConnIdleTime, err := getPositiveDurationEnv(
		"DATABASE_MAX_CONN_IDLE_TIME",
		5*time.Minute,
	)
	if err != nil {
		return Config{}, err
	}

	connectTimeout, err := getPositiveDurationEnv(
		"DATABASE_CONNECT_TIMEOUT",
		5*time.Second,
	)
	if err != nil {
		return Config{}, err
	}

	queryTimeout, err := getPositiveDurationEnv(
		"DATABASE_QUERY_TIMEOUT",
		3*time.Second,
	)
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

	if maxConns <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MAX_CONNS must be positive")
	}

	if minConns <= 0 {
		return Config{}, fmt.Errorf("DATABASE_MIN_CONNS must be positive")
	}

	if minConns > maxConns {
		return Config{}, fmt.Errorf(
			"DATABASE_MIN_CONNS must be less than or equal to DATABASE_MAX_CONNS",
		)
	}

	return Config{
		HTTP: HTTPConfig{
			Addr:              getStringEnv("HTTP_ADDR", ":8080"),
			ReadTimeout:       readTimeout,
			ReadHeaderTimeout: readHeaderTimeout,
			WriteTimeout:      writeTimeout,
			IdleTimeout:       idleTimeout,
		},
		Database: DatabaseConfig{
			URL:             dbURL,
			MaxConns:        maxConns,
			MinConns:        minConns,
			MaxConnLifetime: maxConnLifetime,
			MaxConnIdleTime: maxConnIdleTime,
			ConnectTimeout:  connectTimeout,
			QueryTimeout:    queryTimeout,
		},
		LogLevel:        getStringEnv("LOG_LEVEL", "info"),
		ShutdownTimeout: shutdownTimeout,
	}, nil
}

func getRequiredEnv(name string) (string, error) {
	value := os.Getenv(name)
	if value == "" {
		return "", fmt.Errorf(
			"required environment variable %s is not set",
			name,
		)
	}

	return value, nil
}

func getStringEnv(name, defaultVal string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}

	return defaultVal
}

func getPositiveDurationEnv(
	name string,
	defaultVal time.Duration,
) (time.Duration, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultVal, nil
	}

	duration, err := time.ParseDuration(value)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid duration in %s: %w",
			name,
			err,
		)
	}

	if duration <= 0 {
		return 0, fmt.Errorf(
			"%s must be positive",
			name,
		)
	}

	return duration, nil
}

func getInt32Env(name string, defaultVal int32) (int32, error) {
	value := os.Getenv(name)
	if value == "" {
		return defaultVal, nil
	}

	number, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf(
			"invalid integer in %s: %w",
			name,
			err,
		)
	}

	return int32(number), nil
}
