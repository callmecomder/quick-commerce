package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port  string
	DBDsn string
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	dbHost := envOr("DB_HOST", "localhost")
	dbPort := envOr("DB_PORT", "3306")
	dbUser := envOr("DB_USER", "root")
	dbPass := envOr("DB_PASSWORD", "Pankaj@19721972")
	dbName := envOr("DB_NAME", "quickcommerce")

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPass, dbHost, dbPort, dbName)

	return Config{
		Port:  port,
		DBDsn: dsn,
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
