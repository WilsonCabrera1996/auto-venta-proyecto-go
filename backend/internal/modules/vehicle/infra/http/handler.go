package http

import (
	"net/http"

	"autoventa/backend/internal/infra/http/respond"
	"autoventa/backend/internal/modules/vehicle/domain"
	"autoventa/backend/internal/modules/vehicle/usecase"
)

// Handler traduce HTTP <-> casos de uso. No decide reglas de negocio: si algo
// se rechaza, es porque el dominio lo rechazo.
type Handler struct {
	svc *usecase.Service
}

func NewHandler(svc *usecase.Service) *Handler {
	return &Handler{svc: svc}
}

// Crear godoc
func (h *Handler) Crear(w http.ResponseWriter, r *http.Request) {
	var req vehicleRequest
	if err := respond.DecodeJSON(w, r, &req); err != nil {
		respond.Fail(w, err)
		return
	}
	params, err := req.toParams()
	if err != nil {
		respond.Fail(w, err)
		return
	}
	v, err := h.svc.Crear(r.Context(), params)
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.Created(w, toResponse(v))
}

// Obtener godoc
func (h *Handler) Obtener(w http.ResponseWriter, r *http.Request) {
	v, err := h.svc.Obtener(r.Context(), r.PathValue("id"))
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.OK(w, toResponse(v))
}

// Listar godoc
func (h *Handler) Listar(w http.ResponseWriter, r *http.Request) {
	f, err := parseFiltro(r)
	if err != nil {
		respond.Fail(w, err)
		return
	}
	res, err := h.svc.Listar(r.Context(), f)
	if err != nil {
		respond.Fail(w, err)
		return
	}
	items := make([]vehicleResponse, 0, len(res.Items))
	for _, v := range res.Items {
		items = append(items, toResponse(v))
	}
	respond.OK(w, map[string]any{
		"items":    items,
		"total":    res.Total,
		"page":     res.Page,
		"per_page": res.PerPage,
	})
}

// Actualizar godoc
func (h *Handler) Actualizar(w http.ResponseWriter, r *http.Request) {
	var req vehicleRequest
	if err := respond.DecodeJSON(w, r, &req); err != nil {
		respond.Fail(w, err)
		return
	}
	params, err := req.toParams()
	if err != nil {
		respond.Fail(w, err)
		return
	}
	v, err := h.svc.Actualizar(r.Context(), r.PathValue("id"), params)
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.OK(w, toResponse(v))
}

// ActualizarEstado godoc
func (h *Handler) ActualizarEstado(w http.ResponseWriter, r *http.Request) {
	var req statusRequest
	if err := respond.DecodeJSON(w, r, &req); err != nil {
		respond.Fail(w, err)
		return
	}
	v, err := h.svc.ActualizarEstado(r.Context(), r.PathValue("id"), domain.Status(req.Status))
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.OK(w, toResponse(v))
}

// Eliminar godoc
func (h *Handler) Eliminar(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Eliminar(r.Context(), r.PathValue("id")); err != nil {
		respond.Fail(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
