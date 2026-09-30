// Package logger configura el logger estructurado de la stdlib.
package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init configura slog. En produccion emite JSON porque Render lo indexa como tal;
// en desarrollo emite texto para leerlo mejor en la terminal.
func Init(env, level string) {
	var lv slog.Level
	switch strings.ToLower(level) {
	case "debug":
		lv = slog.LevelDebug
	case "warn":
		lv = slog.LevelWarn
	case "error":
		lv = slog.LevelError
	default:
		lv = slog.LevelInfo
	}
	opts := &slog.HandlerOptions{Level: lv}
	var h slog.Handler
	if env == "production" {
		h = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		h = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(h))
}
