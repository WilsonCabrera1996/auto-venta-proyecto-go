package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"autoventa/backend/internal/infra/http/respond"
	"autoventa/backend/internal/modules/vehicle/usecase"
)

// HandlerReportes expone la conciliacion de ventas. Vive en el modulo vehicle
// porque los datos son vehiculos, aunque la ruta cuelgue de /api/v1/reportes.
type HandlerReportes struct {
	svc *usecase.Service
}

func NewHandlerReportes(svc *usecase.Service) *HandlerReportes {
	return &HandlerReportes{svc: svc}
}

// VentasMensuales godoc
func (h *HandlerReportes) VentasMensuales(w http.ResponseWriter, r *http.Request) {
	anio, mes, err := parseMes(r.URL.Query().Get("mes"))
	if err != nil {
		respond.Fail(w, err)
		return
	}
	rep, err := h.svc.ReporteVentasMensual(r.Context(), anio, mes)
	if err != nil {
		respond.Fail(w, err)
		return
	}
	respond.OK(w, rep)
}

// parseMes acepta "YYYY-MM". Por defecto, el mes en curso.
func parseMes(v string) (int, time.Month, error) {
	ahora := time.Now().UTC()
	if strings.TrimSpace(v) == "" {
		return ahora.Year(), ahora.Month(), nil
	}
	partes := strings.Split(strings.TrimSpace(v), "-")
	if len(partes) != 2 {
		return 0, 0, respond.BadRequest("mes debe tener formato YYYY-MM")
	}
	anio, err := strconv.Atoi(partes[0])
	if err != nil {
		return 0, 0, respond.BadRequest("año invalido")
	}
	m, err := strconv.Atoi(partes[1])
	if err != nil || m < 1 || m > 12 {
		return 0, 0, respond.BadRequest("mes invalido")
	}
	return anio, time.Month(m), nil
}
