package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"autoventa/backend/internal/modules/vehicle/domain"
)

// toParams convierte el request HTTP en los params del dominio, incluyendo el
// parseo del precio decimal a centavos.
func (v vehicleRequest) toParams() (domain.Params, error) {
	priceCents, err := parsePrice(v.Price)
	if err != nil {
		return domain.Params{}, err
	}
	return domain.Params{
		Brand:        v.Brand,
		Model:        v.Model,
		Year:         v.Year,
		PriceCents:   priceCents,
		Status:       domain.Status(strings.ToLower(strings.TrimSpace(v.Status))),
		VIN:          v.VIN,
		Plate:        v.Plate,
		Mileage:      v.Mileage,
		Color:        v.Color,
		FuelType:     v.FuelType,
		Transmission: v.Transmission,
		BodyType:     v.BodyType,
		Doors:        v.Doors,
		Seats:        v.Seats,
		EngineCC:     v.EngineCC,
		Horsepower:   v.Horsepower,
		Drivetrain:   v.Drivetrain,
		Description:  v.Description,
		ImageURL:     v.ImageURL,
	}, nil
}

func toResponse(v domain.Vehicle) vehicleResponse {
	return vehicleResponse{
		ID:           v.ID,
		Brand:        v.Brand,
		Model:        v.Model,
		Year:         v.Year,
		Price:        formatPrice(v.PriceCents),
		PriceCents:   v.PriceCents,
		Status:       string(v.Status),
		VIN:          v.VIN,
		Plate:        v.Plate,
		Mileage:      v.Mileage,
		Color:        v.Color,
		FuelType:     v.FuelType,
		Transmission: v.Transmission,
		BodyType:     v.BodyType,
		Doors:        v.Doors,
		Seats:        v.Seats,
		EngineCC:     v.EngineCC,
		Horsepower:   v.Horsepower,
		Drivetrain:   v.Drivetrain,
		Description:  v.Description,
		ImageURL:     v.ImageURL,
		CreatedAt:    v.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    v.UpdatedAt.Format(time.RFC3339),
	}
}

// parsePrice convierte "45000.00" o "45000,50" en centavos enteros.
// Devuelve 0 sin error cuando el campo viene vacio, para que el dominio
// sea quien emita el error de validacion.
func parsePrice(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	s = strings.ReplaceAll(s, ",", ".")
	// Rechaza separadores de miles y cualquier cosa que no sea numero decimal.
	if strings.Count(s, ".") > 1 {
		return 0, fmt.Errorf("%w: precio con formato invalido", domain.ErrValidation)
	}
	parteEntera, parteDecimal, _ := strings.Cut(s, ".")
	entero, err := strconv.ParseInt(parteEntera, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: precio no numerico", domain.ErrValidation)
	}
	if entero < 0 {
		return 0, fmt.Errorf("%w: precio negativo", domain.ErrValidation)
	}
	var centavos int64
	switch len(parteDecimal) {
	case 0:
	case 1:
		centavos, err = strconv.ParseInt(parteDecimal+"0", 10, 64)
	case 2:
		centavos, err = strconv.ParseInt(parteDecimal, 10, 64)
	default:
		// Mas de 2 decimales se trunca; no es error de formato.
		centavos, err = strconv.ParseInt(parteDecimal[:2], 10, 64)
	}
	if err != nil {
		return 0, fmt.Errorf("%w: precio no numerico", domain.ErrValidation)
	}
	return entero*100 + centavos, nil
}

func formatPrice(cents int64) string {
	return fmt.Sprintf("%d.%02d", cents/100, cents%100)
}

func parseFiltro(r *http.Request) (domain.Filtro, error) {
	q := r.URL.Query()
	f := domain.Filtro{
		Status:  domain.Status(strings.ToLower(q.Get("status"))),
		Brand:   q.Get("brand"),
		Search:  q.Get("search"),
		OrderBy: q.Get("order_by"),
		Page:    atoiDefault(q.Get("page"), 1),
		PerPage: atoiDefault(q.Get("per_page"), 20),
	}
	if q.Get("asc") != "" {
		f.Asc = q.Get("asc") == "true"
	}
	var err error
	if f.MinYear, err = atoiOpt(q.Get("min_year")); err != nil {
		return f, err
	}
	if f.MaxYear, err = atoiOpt(q.Get("max_year")); err != nil {
		return f, err
	}
	if f.MinPrice, err = parsePrice(q.Get("min_price")); err != nil {
		return f, err
	}
	if f.MaxPrice, err = parsePrice(q.Get("max_price")); err != nil {
		return f, err
	}
	return f, nil
}

func atoiDefault(s string, def int) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return def
	}
	return n
}

func atoiOpt(s string) (int, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("%w: parametro numerico invalido", domain.ErrValidation)
	}
	return n, nil
}
