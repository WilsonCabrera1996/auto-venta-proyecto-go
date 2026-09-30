package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
)

// Recovery evita que un panic en un handler tumbe el proceso completo. Es
// imprescindible en Render: un panic sin recover deja el servicio caido hasta
// el siguiente despliegue.
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("panic recuperado",
					"panic", rec,
					"method", r.Method,
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_, _ = w.Write([]byte(`{"message":"error interno","code":"internal"}`))
			}
		}()
		next.ServeHTTP(w, r)
	})
}
