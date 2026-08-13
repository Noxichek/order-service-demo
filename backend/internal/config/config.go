package config

import (
    "os"
)

type Config struct {
    DBDSN     string // строка подключения к PostgreSQL
    KafkaBroker string
    KafkaTopic  string
    HTTPPort    string
}

func Load() *Config {
    return &Config{
        DBDSN:       getEnv("DATABASE_URL", "postgres://orderuser:orderpass@localhost:5432/orders?sslmode=disable"),
        KafkaBroker: getEnv("KAFKA_BROKER", "localhost:9092"),
        KafkaTopic:  getEnv("KAFKA_TOPIC", "orders"),
        HTTPPort:    getEnv("HTTP_PORT", "8081"),
    }
}

func getEnv(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}