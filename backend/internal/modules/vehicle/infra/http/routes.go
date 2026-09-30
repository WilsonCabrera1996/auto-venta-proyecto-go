package http

import (
	"net/http"

	"autoventa/backend/internal/infra/http/middleware"
	"autoventa/backend/internal/modules/session/domain"
)

// Middleware reexportado para no arrastrar el paquete de middleware a quien
// solo necesita la firma.
type Middleware = middleware.Middleware

// RegistrarRutas monta el modulo. Go 1.22+ resuelve metodos y comodines {id} en
// el ServeMux estandar, asi que no hace falta chi ni gin: cero dependencias en
// la capa HTTP.
//
// Cada ruta declara el permiso que exige. El vendedor puede cargar y marcar
// vendido; no puede editar ni borrar. Esa matriz vive en
// session/domain/user.go (permisosPorRol), no aca: aca solo se declara que
// permiso pide cada endpoint.
func RegistrarRutas(mux *http.ServeMux, h *Handler, autenticado Middleware) {
	// Crear: dueño y vendedor.
	crear := autenticado(middleware.RequirePermiso(domain.PermVehiculoCrear)(http.HandlerFunc(h.Crear)))
	mux.Handle("POST /api/v1/vehicles", crear)

	// Listar y obtener: ambos roles necesitan ver el inventario.
	leer := func(fn http.HandlerFunc) http.Handler {
		return autenticado(middleware.RequirePermiso(domain.PermVehiculoVer)(fn))
	}
	mux.Handle("GET /api/v1/vehicles", leer(h.Listar))
	mux.Handle("GET /api/v1/vehicles/{id}", leer(h.Obtener))

	// Editar: solo el dueño. El vendedor carga, no corrige fichas ajenas.
	mux.Handle("PUT /api/v1/vehicles/{id}",
		autenticado(middleware.RequirePermiso(domain.PermVehiculoEditar)(http.HandlerFunc(h.Actualizar))))

	// Marcar vendido: ambos roles. Es la operacion del dia a dia del vendedor.
	mux.Handle("PATCH /api/v1/vehicles/{id}/estado",
		autenticado(middleware.RequirePermiso(domain.PermVehiculoCambiarEstado)(http.HandlerFunc(h.ActualizarEstado))))

	// Borrar: solo el dueño.
	mux.Handle("DELETE /api/v1/vehicles/{id}",
		autenticado(middleware.RequirePermiso(domain.PermVehiculoEliminar)(http.HandlerFunc(h.Eliminar))))
}

// RegistrarRutasReportes monta la conciliacion de ventas. Vive en el modulo
// vehicle porque los datos son vehiculos, aunque la ruta sea /api/v1/reportes.
func RegistrarRutasReportes(mux *http.ServeMux, h *HandlerReportes, autenticado Middleware) {
	mux.Handle("GET /api/v1/reportes/ventas",
		autenticado(middleware.RequirePermiso(domain.PermReporteVentas)(http.HandlerFunc(h.VentasMensuales))))
}
