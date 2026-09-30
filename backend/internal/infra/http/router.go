// Package http arma el router del servicio: middlewares transversales mas las
// rutas de cada modulo. Cada modulo se registra a si mismo con RegistrarRutas.
package http

import (
	"net/http"

	"autoventa/backend/internal/infra/config"
	"autoventa/backend/internal/infra/http/middleware"
	"autoventa/backend/internal/infra/http/respond"
	sessionhttp "autoventa/backend/internal/modules/session/infra/http"
	"autoventa/backend/internal/modules/session/usecase"
	vehiculohttp "autoventa/backend/internal/modules/vehicle/infra/http"
)

// Dependencias son los casos de uso ya construidos. El router no crea nada:
// solo los conecta a las rutas y a los guards.
type Dependencias struct {
	Vehicle  *vehiculohttp.Handler
	Reportes *vehiculohttp.HandlerReportes
	Session  *usecase.Service
}

// NuevoRouter construye el handler raiz.
func NuevoRouter(cfg config.Config, deps Dependencias) http.Handler {
	mux := http.NewServeMux()

	// Autenticacion envuelve las rutas protegidas de los modulos: es lo que
	// impide que un handler vea una peticion sin identidad.
	autenticado := middleware.Autenticar(deps.Session)

	sessionhttp.RegistrarRutas(mux, sessionhttp.NewHandler(deps.Session), deps.Session, autenticado)
	vehiculohttp.RegistrarRutas(mux, deps.Vehicle, autenticado)
	vehiculohttp.RegistrarRutasReportes(mux, deps.Reportes, autenticado)

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		respond.OK(w, map[string]any{"status": "ok", "env": cfg.Env})
	})

	// El orden importa: Recovery va primero para atrapar cualquier panic de los
	// que vienen despues; CORS antes de los middlewares de log para que tambien
	// registre los preflight.
	return middleware.Chain(mux,
		middleware.Recovery,
		middleware.RequestID,
		middleware.CORS(cfg.CORSAllowedOrigins),
		middleware.MaxBodyKey(cfg.MaxBodyBytes),
		middleware.Logger,
	)
}
