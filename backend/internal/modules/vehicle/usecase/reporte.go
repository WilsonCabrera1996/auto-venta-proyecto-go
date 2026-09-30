package usecase

import (
	"context"
	"fmt"
	"time"

	"autoventa/backend/internal/modules/vehicle/domain"
)

// ReporteVentas es la conciliacion de un periodo. Solo mira vehiculos vendidos:
// un vehiculo en stock no es venta.
type ReporteVentas struct {
	Desde          time.Time `json:"desde"`
	Hasta          time.Time `json:"hasta"`
	Unidades       int64     `json:"unidades"`
	IngresosCents  int64     `json:"ingresos_cents"`
	Ingresos       string    `json:"ingresos"`
	PrecioPromedio string    `json:"precio_promedio"`
}

// ReporteVentasMensual calcula el reporte de un mes natural. El periodo se
// calcula en UTC porque CreatedAt se guarda en UTC; mezclar zonas haria que un
// vehiculo vendido a las 23:30 local caiga en el mes equivocado.
func (s *Service) ReporteVentasMensual(ctx context.Context, anio int, mes time.Month) (ReporteVentas, error) {
	if mes < time.January || mes > time.December {
		return ReporteVentas{}, fmt.Errorf("mes invalido: %d", mes)
	}
	desde := time.Date(anio, mes, 1, 0, 0, 0, 0, time.UTC)
	hasta := desde.AddDate(0, 1, 0)

	res, err := s.repo.List(ctx, domain.Filtro{
		Status:  domain.StatusSold,
		Desde:   desde,
		Hasta:   hasta.Add(-time.Nanosecond),
		Page:    1,
		PerPage: 1, // solo necesitamos el agregado, no los items
	})
	if err != nil {
		return ReporteVentas{}, err
	}

	rep := ReporteVentas{
		Desde:         desde,
		Hasta:         hasta,
		Unidades:      res.Total,
		IngresosCents: res.TotalPrecioCents,
		Ingresos:      formatCents(res.TotalPrecioCents),
	}
	if res.Total > 0 {
		rep.PrecioPromedio = formatCents(res.TotalPrecioCents / res.Total)
	} else {
		rep.PrecioPromedio = "0.00"
	}
	return rep, nil
}

func formatCents(c int64) string {
	return fmt.Sprintf("%d.%02d", c/100, c%100)
}
