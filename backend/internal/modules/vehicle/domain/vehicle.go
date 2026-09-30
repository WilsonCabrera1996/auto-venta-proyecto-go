package domain

import (
	"fmt"
	"strings"
	"time"

	"autoventa/backend/internal/platform/id"
)

// Status es el estado comercial de un vehiculo. Es un tipo propio (no string)
// para que la maquina de estados sea explicita y no se pueda comparar por error.
type Status string

const (
	StatusAvailable Status = "available"
	StatusReserved  Status = "reserved"
	StatusSold      Status = "sold"
)

// transiciones permitidas. Se declara completa y explicita aunque hoy todo sea
// permisible, para que endurecer la politica sea cambiar una linea y no reescribir
// codigo. La unica transicion restringida es sold -> available (reingreso).
var transiciones = map[Status][]Status{
	StatusAvailable: {StatusReserved, StatusSold},
	StatusReserved:  {StatusAvailable, StatusSold},
	StatusSold:      {},
}

func (s Status) Valid() bool {
	switch s {
	case StatusAvailable, StatusReserved, StatusSold:
		return true
	}
	return false
}

func (s Status) CanTransitionTo(destino Status) bool {
	for _, permitido := range transiciones[s] {
		if permitido == destino {
			return true
		}
	}
	return false
}

// Vehicle es la entidad del modulo. Los tags json son el contrato publico de la
// entidad en este repo; si mañana el dominio se serializa para otra cosa, se
// revisa aqui y no se toca la logica.
type Vehicle struct {
	ID           string
	Brand        string
	Model        string
	Year         int
	PriceCents   int64
	Status       Status
	VIN          string
	Plate        string
	Mileage      int64
	Color        string
	FuelType     string
	Transmission string
	BodyType     string
	Doors        int
	Seats        int
	EngineCC     int
	Horsepower   int
	Drivetrain   string
	Description  string
	ImageURL     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Params son los datos de entrada de un alta. Separate de Vehicle para que el
// ID y los timestamps los fije el dominio, no el llamador.
type Params struct {
	Brand        string
	Model        string
	Year         int
	PriceCents   int64
	Status       Status
	VIN          string
	Plate        string
	Mileage      int64
	Color        string
	FuelType     string
	Transmission string
	BodyType     string
	Doors        int
	Seats        int
	EngineCC     int
	Horsepower   int
	Drivetrain   string
	Description  string
	ImageURL     string
}

const (
	anioMinimo      = 1886 // primer automovil de la historia
	anioMaximoDelta = 1    // se admite el anio siguiente (vehiculo a la venta)
	maxPuertas      = 6
	maxAsientos     = 9
	vinLongitud     = 17
)

// New valida los invariantes y devuelve la entidad ya normalizada. Todo alta que
// no pase por aqui es un bug, no un dato sucio en la base.
func New(p Params) (Vehicle, error) {
	if p.Status == "" {
		p.Status = StatusAvailable
	}
	v := Vehicle{
		ID:           id.New(),
		Brand:        strings.TrimSpace(p.Brand),
		Model:        strings.TrimSpace(p.Model),
		Year:         p.Year,
		PriceCents:   p.PriceCents,
		Status:       p.Status,
		VIN:          strings.ToUpper(strings.TrimSpace(p.VIN)),
		Plate:        strings.ToUpper(strings.TrimSpace(p.Plate)),
		Mileage:      p.Mileage,
		Color:        strings.TrimSpace(p.Color),
		FuelType:     strings.ToLower(strings.TrimSpace(p.FuelType)),
		Transmission: strings.ToLower(strings.TrimSpace(p.Transmission)),
		BodyType:     strings.ToLower(strings.TrimSpace(p.BodyType)),
		Doors:        p.Doors,
		Seats:        p.Seats,
		EngineCC:     p.EngineCC,
		Horsepower:   p.Horsepower,
		Drivetrain:   strings.ToUpper(strings.TrimSpace(p.Drivetrain)),
		Description:  strings.TrimSpace(p.Description),
		ImageURL:     strings.TrimSpace(p.ImageURL),
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    v0(p),
	}
	if err := v.validate(); err != nil {
		return Vehicle{}, err
	}
	return v, nil
}

func v0(Params) time.Time { return time.Now().UTC() }

func (v Vehicle) validate() error {
	var problemas []string

	if v.Brand == "" {
		problemas = append(problemas, "marca es obligatoria")
	}
	if v.Model == "" {
		problemas = append(problemas, "modelo es obligatorio")
	}
	if v.Year < anioMinimo || v.Year > time.Now().Year()+anioMaximoDelta {
		problemas = append(problemas, fmt.Sprintf("anio fuera de rango (%d-%d)", anioMinimo, time.Now().Year()+anioMaximoDelta))
	}
	if v.PriceCents <= 0 {
		problemas = append(problemas, "precio debe ser mayor a cero")
	}
	if v.Mileage < 0 {
		problemas = append(problemas, "kilometraje no puede ser negativo")
	}
	if !v.Status.Valid() {
		problemas = append(problemas, fmt.Sprintf("estado invalido: %q", v.Status))
	}
	if v.VIN != "" && !validVIN(v.VIN) {
		problemas = append(problemas, "VIN invalido: debe tener 17 caracteres y no incluir I, O ni Q")
	}
	if v.Doors < 0 || v.Doors > maxPuertas {
		problemas = append(problemas, fmt.Sprintf("puertas fuera de rango (0-%d)", maxPuertas))
	}
	if v.Seats < 0 || v.Seats > maxAsientos {
		problemas = append(problemas, fmt.Sprintf("asientos fuera de rango (0-%d)", maxAsientos))
	}
	if v.EngineCC < 0 {
		problemas = append(problemas, "cilindrada no puede ser negativa")
	}
	if v.Horsepower < 0 {
		problemas = append(problemas, "potencia no puede ser negativa")
	}

	if len(problemas) > 0 {
		return fmt.Errorf("%w: %s", ErrValidation, strings.Join(problemas, "; "))
	}
	return nil
}

func validVIN(vin string) bool {
	if len(vin) != vinLongitud {
		return false
	}
	for _, c := range vin {
		switch c {
		case 'I', 'O', 'Q':
			return false
		}
		if c != '-' && !strings.ContainsRune("0123456789ABCDEFGHJKLMNPRSTUVWXYZ", c) {
			return false
		}
	}
	return true
}

// TransitionTo aplica la maquina de estados. Es el unico camino para cambiar Status,
// incluso desde un caso de uso de update completo.
func (v *Vehicle) TransitionTo(destino Status) error {
	if !destino.Valid() {
		return fmt.Errorf("%w: estado destino invalido %q", ErrInvalidTransition, destino)
	}
	if v.Status == destino {
		return nil
	}
	if !v.Status.CanTransitionTo(destino) {
		return fmt.Errorf("%w: no se puede pasar de %q a %q", ErrInvalidTransition, v.Status, destino)
	}
	v.Status = destino
	return nil
}

func (v Vehicle) Disponible() bool { return v.Status == StatusAvailable }
