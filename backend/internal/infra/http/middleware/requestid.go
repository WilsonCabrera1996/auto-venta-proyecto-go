package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
)

type ctxKey int

const requestIDKey ctxKey = iota

// RequestID adjunta un identificador a cada peticion para poder correlacionar
// logs cuando hay varias peticiones simultaneas.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = nuevoID()
		}
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDKey, rid)))
	})
}

// RequestIDFrom extrae el id del contexto.
func RequestIDFrom(ctx context.Context) string {
	v, _ := ctx.Value(requestIDKey).(string)
	return v
}

func nuevoID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "desconocido"
	}
	return hex.EncodeToString(b[:])
}
