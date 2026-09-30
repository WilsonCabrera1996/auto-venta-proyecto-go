// Package config lee la configuracion del entorno. Cada servicio tiene su
// propio .env porque cada uno corre en un host distinto; no hay archivo
// compartido que valga.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	sessiondomain "autoventa/backend/internal/modules/session/domain"
)

// Config es inmutable una vez construida.
type Config struct {
	Port               string
	Env                string
	CORSAllowedOrigins []string
	MaxBodyBytes       int64
	LogLevel           string
	SessionTTL         time.Duration
	OwnerEmail         string
	OwnerPassword      string
}

// Load arma la Config desde el entorno y aplica defaults pensados para que local
// y Render se comporten igual.
func Load() Config {
	return Config{
		Port:               getEnv("PORT", "8080"),
		Env:                getEnv("APP_ENV", "development"),
		CORSAllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "")),
		MaxBodyBytes:       getEnvInt64("MAX_BODY_BYTES", 1<<20),
		LogLevel:           getEnv("LOG_LEVEL", "info"),
		SessionTTL:         getEnvDuration("SESSION_TTL", sessiondomain.TTLDefault),
		OwnerEmail:         getEnv("OWNER_EMAIL", ""),
		OwnerPassword:      os.Getenv("OWNER_PASSWORD"),
	}
}

// IsProduction indica si estamos en el deploy real.
func (c Config) IsProduction() bool { return c.Env == "production" }

func getEnv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

func getEnvDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}

func getEnvInt64(key string, def int64) int64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func splitCSV(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
