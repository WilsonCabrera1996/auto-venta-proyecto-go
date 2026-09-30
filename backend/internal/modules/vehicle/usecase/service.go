package usecase

import (
	"context"

	"autoventa/backend/internal/modules/vehicle/domain"
)

// Service orquesta los casos de uso del modulo vehiculo. Solo conoce el puerto
// del dominio: no sabe si el repo es memoria o Supabase, ni que existe HTTP.
type Service struct {
	repo domain.Repository
	// Reloj y generador de id se inyectan para que los tests sean deterministas.
	// El id lo genera el dominio; aqui solo se expone el repo.
}

func NewService(repo domain.Repository) *Service {
	return &Service{repo: repo}
}

// Crear registra un vehiculo. El ID lo asigna el dominio.
func (s *Service) Crear(ctx context.Context, p domain.Params) (domain.Vehicle, error) {
	v, err := domain.New(p)
	if err != nil {
		return domain.Vehicle{}, err
	}
	// Uniqueness check explicito para poder devolver 409 en vez de 500 cuando
	// la restriccion UNIQUE de Postgres reviente.
	if v.VIN != "" {
		if _, err := s.repo.GetByVIN(ctx, v.VIN); err == nil {
			return domain.Vehicle{}, domain.ErrAlreadyExists
		}
	}
	return s.repo.Create(ctx, v)
}

// Obtener devuelve un vehiculo por id.
func (s *Service) Obtener(ctx context.Context, id string) (domain.Vehicle, error) {
	if id == "" {
		return domain.Vehicle{}, domain.ErrNotFound
	}
	return s.repo.GetByID(ctx, id)
}

// Actualizar reemplaza los campos del vehiculo. Si se envia status, pasa por la
// maquina de estados; si no, conserva el actual.
func (s *Service) Actualizar(ctx context.Context, id string, p domain.Params) (domain.Vehicle, error) {
	actual, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Vehicle{}, err
	}
	// Se reaplica el constructor sobre los params del cliente para revalidar y
	// renormalizar (trim, upper) con las mismas reglas del alta.
	actualizado, err := domain.New(p)
	if err != nil {
		return domain.Vehicle{}, err
	}
	if p.Status == "" {
		actualizado.Status = actual.Status
	} else if err := actual.TransitionTo(actualizado.Status); err != nil {
		return domain.Vehicle{}, err
	}
	actualizado.ID = actual.ID
	actualizado.CreatedAt = actual.CreatedAt

	if actualizado.VIN != "" && actualizado.VIN != actual.VIN {
		if _, err := s.repo.GetByVIN(ctx, actualizado.VIN); err == nil {
			return domain.Vehicle{}, domain.ErrAlreadyExists
		}
	}
	return s.repo.Update(ctx, actualizado)
}

// ActualizarEstado es la operacion de la maquina de estados expuesta aparte,
// porque el dashboard la invoca con frecuencia y merece su propio caso de uso.
func (s *Service) ActualizarEstado(ctx context.Context, id string, destino domain.Status) (domain.Vehicle, error) {
	v, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return domain.Vehicle{}, err
	}
	if err := v.TransitionTo(destino); err != nil {
		return domain.Vehicle{}, err
	}
	return s.repo.Update(ctx, v)
}

// Eliminar borra un vehiculo por id.
func (s *Service) Eliminar(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

// Listar devuelve una pagina filtrada.
func (s *Service) Listar(ctx context.Context, f domain.Filtro) (domain.Listado, error) {
	if f.Page < 1 {
		f.Page = 1
	}
	if f.PerPage < 1 {
		f.PerPage = 20
	}
	if f.PerPage > 100 {
		f.PerPage = 100
	}
	return s.repo.List(ctx, f)
}
