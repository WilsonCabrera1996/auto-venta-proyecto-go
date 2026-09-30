package middleware

import (
	"context"
	"net/http"
	"strings"

	"autoventa/backend/internal/infra/http/respond"
	"autoventa/backend/internal/modules/session/domain"
)

// ctxKey vive en requestid.go; aco solo se declaran claves distintas. El +100
// evita colisionar con requestIDKey, que es iota del primer bloque.
const (
	actorKey ctxKey = 100 + iota
	tokenKey
)

// Actor es el caso de uso que resuelve tokens. Es una interfaz y no el
// *usecase.Service concreto para que el middleware no dependa del modulo.
type Resolver interface {
	ResolverAutenticado(ctx context.Context, token string) (domain.User, error)
}

// Autenticar valida el Bearer token y guarda el actor en el contexto. Si el
// token no sirve devuelve 401 inmediatamente.
func Autenticar(res Resolver) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearer(r)
			if token == "" {
				respond.Fail(w, domain.ErrNoAutorizado)
				return
			}
			actor, err := res.ResolverAutenticado(r.Context(), token)
			if err != nil {
				respond.Fail(w, err)
				return
			}
			ctx := context.WithValue(r.Context(), actorKey, actor)
			ctx = context.WithValue(ctx, tokenKey, token)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequirePermiso es el guard de autorizacion. Verifica un PERMISO, no un rol:
// los roles son un detalle del dominio, y asi el middleware no necesita
// conocer la matriz de permisos.
func RequirePermiso(p domain.Permiso) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			actor, ok := Actor(r.Context())
			if !ok {
				respond.Fail(w, domain.ErrNoAutorizado)
				return
			}
			if !actor.Role.Puede(p) {
				respond.Fail(w, domain.ErrProhibido)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// Actor recupera la identidad del contexto. Devuelve false si no hay.
func Actor(ctx context.Context) (domain.User, bool) {
	u, ok := ctx.Value(actorKey).(domain.User)
	return u, ok
}

func bearer(r *http.Request) string {
	h := r.Header.Get("Authorization")
	const prefijo = "Bearer "
	if len(h) > len(prefijo) && strings.EqualFold(h[:len(prefijo)], prefijo) {
		return strings.TrimSpace(h[len(prefijo):])
	}
	return ""
}

// TokenDe recupera el token en claro del contexto. Lo necesita el logout para
// revocar exactamente esa sesion y no otra.
func TokenDe(ctx context.Context) string {
	t, _ := ctx.Value(tokenKey).(string)
	return t
}
