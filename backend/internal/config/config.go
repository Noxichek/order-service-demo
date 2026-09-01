package config

import "os"

type Config struct {
    DBDSN          string
    KafkaBroker    string
    KafkaTopic     string
    HTTPPort       string
    AuthToken      string
    AllowedOrigin  string
    BindAddress    string
}

func Load() *Config {
    return &Config{
        DBDSN:         getEnv("DATABASE_URL", "postgres://orderuser:orderpass@localhost:5432/orders?sslmode=disable"),
        KafkaBroker:   getEnv("KAFKA_BROKER", "localhost:9092"),
        KafkaTopic:    getEnv("KAFKA_TOPIC", "orders"),
        HTTPPort:      getEnv("HTTP_PORT", "8081"),
        AuthToken:     getEnv("AUTH_TOKEN", "devtoken"),
        AllowedOrigin: getEnv("ALLOWED_ORIGIN", "http://localhost:4200"),
        BindAddress:   getEnv("BIND_ADDRESS", "127.0.0.1"),
    }
}

func getEnv(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}
