package middleware

import (
	"net/http"
	"strings"
)

// CORS es obligatorio en esta arquitectura: el frontend vive en Vercel y el
// backend en Render, o sea origenes distintos. Sin esto el navegador bloquea
// cada llamada antes de que llegue al backend.
//
// origins vacio significa "sin CORS", util para tests y para curl. Se comparan
// contra el Origin de la peticion porque Render sirve por su propio dominio y
// se necesita permitir explicitamente el dominio del frontend.
func CORS(origins []string) Middleware {
	permitidos := make(map[string]bool, len(origins))
	for _, o := range origins {
		o = strings.TrimSpace(o)
		if o != "" {
			permitidos[o] = true
		}
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origen := r.Header.Get("Origin")
			if permitidos[origen] {
				w.Header().Set("Access-Control-Allow-Origin", origen)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
				w.Header().Set("Access-Control-Max-Age", "86400")
				w.Header().Set("Vary", "Origin")
			}
			if r.Method == http.MethodOptions {
				// Preflight siempre 204; si el origen no esta permitido, igual
				// se corta aqui para que el navegador no ni siquiera intente.
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// MaxBodyKey limita el tamano del cuerpo para no agotar los 512MB de RAM.
func MaxBodyKey(limiteBytes int64) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.ContentLength > limiteBytes {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusRequestEntityTooLarge)
				_, _ = w.Write([]byte(`{"message":"cuerpo demasiado grande","code":"payload_too_large"}`))
				return
			}
			r.Body = http.MaxBytesReader(w, r.Body, limiteBytes)
			next.ServeHTTP(w, r)
		})
	}
}
