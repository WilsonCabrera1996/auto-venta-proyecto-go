// Package repos contiene las implementaciones de los puertos del dominio.
// La de memoria sirve para tests internos y para levantar el servicio sin base
// de datos; la de Supabase vendra con .env y las migraciones.
package repos

import (
	"context"
	"sort"
	"strings"
	"sync"

	"autoventa/backend/internal/modules/vehicle/domain"
)

// VehicleMemory es un repositorio en memoria, seguro para uso concurrente.
type VehicleMemory struct {
	mu   sync.RWMutex
	byID map[string]domain.Vehicle
	// vin -> id, indice secundario para no recorrer todo el mapa en cada
	// busqueda por VIN, que es la unica unicidad real del dominio.
	byVIN map[string]string
}

func NewVehicleMemory() *VehicleMemory {
	return &VehicleMemory{
		byID:  make(map[string]domain.Vehicle),
		byVIN: make(map[string]string),
	}
}

func (r *VehicleMemory) Create(_ context.Context, v domain.Vehicle) (domain.Vehicle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, existe := r.byID[v.ID]; existe {
		return domain.Vehicle{}, domain.ErrAlreadyExists
	}
	r.byID[v.ID] = v
	if v.VIN != "" {
		r.byVIN[v.VIN] = v.ID
	}
	return v, nil
}

func (r *VehicleMemory) Update(_ context.Context, v domain.Vehicle) (domain.Vehicle, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	previo, existe := r.byID[v.ID]
	if !existe {
		return domain.Vehicle{}, domain.ErrNotFound
	}
	vinPrevio, _ := r.byVIN[previo.VIN]
	r.byID[v.ID] = v
	if vinPrevio != "" && vinPrevio != v.VIN {
		delete(r.byVIN, vinPrevio)
	}
	if v.VIN != "" {
		r.byVIN[v.VIN] = v.ID
	}
	return v, nil
}

func (r *VehicleMemory) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	v, existe := r.byID[id]
	if !existe {
		return domain.ErrNotFound
	}
	if v.VIN != "" {
		delete(r.byVIN, v.VIN)
	}
	delete(r.byID, id)
	return nil
}

func (r *VehicleMemory) GetByID(_ context.Context, id string) (domain.Vehicle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	v, ok := r.byID[id]
	if !ok {
		return domain.Vehicle{}, domain.ErrNotFound
	}
	return v, nil
}

func (r *VehicleMemory) GetByVIN(_ context.Context, vin string) (domain.Vehicle, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byVIN[strings.ToUpper(vin)]
	if !ok {
		return domain.Vehicle{}, domain.ErrNotFound
	}
	return r.byID[id], nil
}

func (r *VehicleMemory) List(_ context.Context, f domain.Filtro) (domain.Listado, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]domain.Vehicle, 0, len(r.byID))
	for _, v := range r.byID {
		if cumple(v, f) {
			items = append(items, v)
		}
	}

	ordenar(items, f)

	total := int64(len(items))
	var suma int64
	for _, v := range items {
		suma += v.PriceCents
	}
	desde := (f.Page - 1) * f.PerPage
	if desde >= len(items) {
		return domain.Listado{
			Items: []domain.Vehicle{}, Total: total, Page: f.Page,
			PerPage: f.PerPage, TotalPrecioCents: suma,
		}, nil
	}
	hasta := desde + f.PerPage
	if hasta > len(items) {
		hasta = len(items)
	}
	return domain.Listado{
		Items:            items[desde:hasta],
		Total:            total,
		Page:             f.Page,
		PerPage:          f.PerPage,
		TotalPrecioCents: suma,
	}, nil
}

func cumple(v domain.Vehicle, f domain.Filtro) bool {
	if f.Status != "" && v.Status != f.Status {
		return false
	}
	if f.Brand != "" && !strings.EqualFold(v.Brand, f.Brand) {
		return false
	}
	if f.Search != "" {
		q := strings.ToLower(f.Search)
		if !strings.Contains(strings.ToLower(v.Brand), q) &&
			!strings.Contains(strings.ToLower(v.Model), q) &&
			!strings.Contains(strings.ToLower(v.Description), q) {
			return false
		}
	}
	if f.MinYear > 0 && v.Year < f.MinYear {
		return false
	}
	if f.MaxYear > 0 && v.Year > f.MaxYear {
		return false
	}
	if f.MinPrice > 0 && v.PriceCents < f.MinPrice {
		return false
	}
	if f.MaxPrice > 0 && v.PriceCents > f.MaxPrice {
		return false
	}
	if !f.Desde.IsZero() && v.CreatedAt.Before(f.Desde) {
		return false
	}
	if !f.Hasta.IsZero() && v.CreatedAt.After(f.Hasta) {
		return false
	}
	return true
}

func ordenar(items []domain.Vehicle, f domain.Filtro) {
	clave := f.OrderBy
	if clave == "" {
		clave = "created_at"
	}
	desc := !f.Asc
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		var menor bool
		switch clave {
		case "price":
			menor = a.PriceCents < b.PriceCents
		case "year":
			menor = a.Year < b.Year
		case "brand":
			menor = strings.ToLower(a.Brand) < strings.ToLower(b.Brand)
		default:
			menor = a.CreatedAt.Before(b.CreatedAt)
		}
		if desc {
			return !menor
		}
		return menor
	})
}
