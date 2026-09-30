package http

import (
	"net/http"
	"time"

	"autoventa/backend/internal/infra/http/middleware"
	"autoventa/backend/internal/infra/http/respond"
	"autoventa/backend/internal/modules/session/domain"
	"autoventa/backend/internal/modules/session/usecase"
)

// Handler HTTP del modulo. Igual que vehicle: traduce, no decide. Quien decide
// si la llamada se permite es el middleware, antes de llegar aca.
type Handler struct {
	svc *usecase.Service
}

func NewHandler(svc *usecase.Service) *Handler { return &Handler{svc: svc} }

// Login godoc
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := respond.DecodeJSON(w, r, &req); err != nil {
		respond.Fail(w, err)
		return
	}
	user, token, expira, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.OK(w, loginResponse{
		Token:     token,
		ExpiresAt: expira.Format(time.RFC3339),
		User:      toResponse(user),
	})
}

// Logout godoc
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := middleware.TokenDe(r.Context())
	if err := h.svc.Logout(r.Context(), token); err != nil {
		respond.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Yo godoc
func (h *Handler) Yo(w http.ResponseWriter, r *http.Request) {
	actor, err := h.svc.Yo(r.Context(), actorDe(r))
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.OK(w, toResponse(actor))
}

// CrearUsuario godoc
func (h *Handler) CrearUsuario(w http.ResponseWriter, r *http.Request) {
	var req crearUsuarioRequest
	if err := respond.DecodeJSON(w, r, &req); err != nil {
		respond.Fail(w, err)
		return
	}
	u, err := h.svc.CrearUsuario(r.Context(), req.Email, req.Nombre, req.Password, domain.Role(req.Role))
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.Created(w, toResponse(u))
}

// ListarUsuarios godoc
func (h *Handler) ListarUsuarios(w http.ResponseWriter, r *http.Request) {
	usuarios, err := h.svc.ListarUsuarios(r.Context())
	if err != nil {
		respond.Fail(w, err)
		return
	}
	out := make([]userResponse, 0, len(usuarios))
	for _, u := range usuarios {
		out = append(out, toResponse(u))
	}
	respond.OK(w, map[string]any{"items": out})
}

// CambiarRol godoc
func (h *Handler) CambiarRol(w http.ResponseWriter, r *http.Request) {
	var req cambiarRolRequest
	if err := respond.DecodeJSON(w, r, &req); err != nil {
		respond.Fail(w, err)
		return
	}
	actor := actorDe(r)
	u, err := h.svc.CambiarRol(r.Context(), actor, r.PathValue("id"), domain.Role(req.Role))
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.OK(w, toResponse(u))
}

// Desactivar godoc
func (h *Handler) Desactivar(w http.ResponseWriter, r *http.Request) {
	actor := actorDe(r)
	if err := h.svc.Desactivar(r.Context(), actor, r.PathValue("id")); err != nil {
		respond.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// actorDe recupera la identidad que Autenticar dejo en el contexto. Si no esta,
// la peticion no paso por el middleware y no hay nada que hacer aca: se deja que
// el usecase lo rechace.
func actorDe(r *http.Request) domain.User {
	u, _ := middleware.Actor(r.Context())
	return u
}

// toResponse arma el DTO publico. Los permisos viajan explicitos para que el
// frontend pueda ocultar botones en vez de adivinar por el rol.
func toResponse(u domain.User) userResponse {
	concedidos := domain.PermisosDe(u.Role)
	permisos := make([]string, 0, len(concedidos))
	for _, p := range concedidos {
		permisos = append(permisos, string(p))
	}
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		Nombre:    u.Nombre,
		Role:      string(u.Role),
		Activo:    u.Activo,
		Permisos:  permisos,
		CreatedAt: u.CreatedAt.Format(time.RFC3339),
	}
}
