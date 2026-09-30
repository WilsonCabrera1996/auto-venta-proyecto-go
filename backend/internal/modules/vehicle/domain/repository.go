package domain

import (
	"context"
	"time"
)

// Filtro son los criterios de busqueda del listado. Todos los campos son
// opcionales; el repositorio ignora los vacios.
type Filtro struct {
	Status   Status
	Brand    string
	Search   string
	MinYear  int
	MaxYear  int
	MinPrice int64
	MaxPrice int64
	OrderBy  string
	Asc      bool
	Page     int
	PerPage  int
	// Desde y Hasta acotan por CreatedAt. Se usan para conciliar ventas de un
	// periodo sin tener que traer el inventario completo.
	Desde time.Time
	Hasta time.Time
}

// Listado es la pagina de resultados que devuelve el repositorio.
type Listado struct {
	Items   []Vehicle
	Total   int64
	Page    int
	PerPage int
	// TotalPrecioCents suma los precios de TODOS los que pasan el filtro, no
	// solo los de la pagina. Es lo que permite conciliar ventas sin paginar.
	TotalPrecioCents int64
}

// Repository es el puerto de salida del modulo. El dominio define la interfaz;
// infra implementa (hoy en memoria, luego Supabase). Los casos de uso solo
// dependen de esta firma.
type Repository interface {
	Create(ctx context.Context, v Vehicle) (Vehicle, error)
	Update(ctx context.Context, v Vehicle) (Vehicle, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (Vehicle, error)
	GetByVIN(ctx context.Context, vin string) (Vehicle, error)
	List(ctx context.Context, f Filtro) (Listado, error)
}
