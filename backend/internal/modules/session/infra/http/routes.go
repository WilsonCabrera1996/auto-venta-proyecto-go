package http

import (
	"net/http"

	"autoventa/backend/internal/infra/http/middleware"
	"autoventa/backend/internal/modules/session/domain"
)

// Middleware es el tipo que usan los guards, reexportado para que quien llama
// no tenga que importar el paquete de middleware solo por la firma.
type Middleware = middleware.Middleware

// RegistrarRutas monta el modulo. Cada ruta declara su propia exigencia de
// permiso en el sitio donde se lee, en vez de un switch central que hay que
// acordarse de actualizar cada vez que se agrega un endpoint.
func RegistrarRutas(
	mux *http.ServeMux,
	h *Handler,
	res middleware.Resolver,
	autenticado Middleware,
) {
	// Publica: sin sesion no se puede abrir sesion.
	mux.HandleFunc("POST /api/v1/session", h.Login)

	// Autenticada: basta con un token valido.
	mux.Handle("POST /api/v1/session/logout", autenticado(http.HandlerFunc(h.Logout)))
	mux.Handle("GET /api/v1/session/yo", autenticado(http.HandlerFunc(h.Yo)))

	// Administracion de usuarios: solo el dueño (PermUsuarioAdministrar).
	admin := func(p domain.Permiso, fn http.HandlerFunc) http.Handler {
		return autenticado(middleware.RequirePermiso(p)(fn))
	}
	mux.Handle("POST /api/v1/usuarios",
		admin(domain.PermUsuarioAdministrar, h.CrearUsuario))
	mux.Handle("GET /api/v1/usuarios",
		admin(domain.PermUsuarioAdministrar, h.ListarUsuarios))
	mux.Handle("PATCH /api/v1/usuarios/{id}/rol",
		admin(domain.PermUsuarioAdministrar, h.CambiarRol))
	mux.Handle("DELETE /api/v1/usuarios/{id}",
		admin(domain.PermUsuarioAdministrar, h.Desactivar))
}
