package usecase_test

import (
	"context"
	"errors"
	"testing"

	"autoventa/backend/internal/modules/vehicle/domain"
	"autoventa/backend/internal/modules/vehicle/usecase"
	"autoventa/backend/internal/repos"
)

// Los casos de uso se prueban contra el repositorio en memoria: son tests
// internos, sin base de datos ni HTTP. Cuando exista el repo de Postgres, la
// suite sigue valiendo porque el contrato (domain.Repository) no cambia.
func nuevoSvc() *usecase.Service {
	return usecase.NewService(repos.NewVehicleMemory())
}

func paramsBase() domain.Params {
	return domain.Params{
		Brand:      "Toyota",
		Model:      "Corolla",
		Year:       2020,
		PriceCents: 1_850_000,
	}
}

func TestCrearYObtener(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	creado, err := svc.Crear(ctx, paramsBase())
	if err != nil {
		t.Fatalf("Crear() error: %v", err)
	}
	if creado.ID == "" {
		t.Fatal("se esperaba ID asignado")
	}

	obtenido, err := svc.Obtener(ctx, creado.ID)
	if err != nil {
		t.Fatalf("Obtener() error: %v", err)
	}
	if obtenido.ID != creado.ID || obtenido.Brand != "Toyota" {
		t.Errorf("se devolvio otro vehiculo: %+v", obtenido)
	}
}

func TestObtenerInexistente(t *testing.T) {
	svc := nuevoSvc()
	_, err := svc.Obtener(context.Background(), "no-existe")
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("se esperaba ErrNotFound, dio %v", err)
	}
}

func TestCrear_rechazaParamsInvalidosSinTocarElRepo(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	p := paramsBase()
	p.Year = 1500
	if _, err := svc.Crear(ctx, p); !errors.Is(err, domain.ErrValidation) {
		t.Fatalf("se esperaba ErrValidation, dio %v", err)
	}

	res, err := svc.Listar(ctx, domain.Filtro{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Total != 0 {
		t.Errorf("un alta invalida no deberia persistir nada, hay %d", res.Total)
	}
}

func TestCrear_rechazaVINDuplicado(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	p := paramsBase()
	p.VIN = "1HGCM82633A004352"
	if _, err := svc.Crear(ctx, p); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Crear(ctx, p); !errors.Is(err, domain.ErrAlreadyExists) {
		t.Errorf("se esperaba ErrAlreadyExists, dio %v", err)
	}
}

func TestActualizar_revalidaYPreservaCreatedAt(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	creado, _ := svc.Crear(ctx, paramsBase())

	p := paramsBase()
	p.Model = "Camry"
	p.PriceCents = 2_400_000
	actualizado, err := svc.Actualizar(ctx, creado.ID, p)
	if err != nil {
		t.Fatalf("Actualizar() error: %v", err)
	}
	if actualizado.Model != "Camry" {
		t.Errorf("modelo no actualizado: %q", actualizado.Model)
	}
	if actualizado.PriceCents != 2_400_000 {
		t.Errorf("precio no actualizado: %d", actualizado.PriceCents)
	}
	if !actualizado.CreatedAt.Equal(creado.CreatedAt) {
		t.Error("CreatedAt debe preservarse en una actualizacion")
	}
	if actualizado.ID != creado.ID {
		t.Error("el ID debe preservarse")
	}
}

func TestActualizar_sinStatusConservaElActual(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	creado, _ := svc.Crear(ctx, paramsBase())
	if _, err := svc.ActualizarEstado(ctx, creado.ID, domain.StatusSold); err != nil {
		t.Fatal(err)
	}

	// Update sin campo status: no debe resucitar el vehiculo vendido.
	p := paramsBase()
	p.Model = "Corolla 2.0"
	actualizado, err := svc.Actualizar(ctx, creado.ID, p)
	if err != nil {
		t.Fatal(err)
	}
	if actualizado.Status != domain.StatusSold {
		t.Errorf("se esperaba conservar sold, dio %q", actualizado.Status)
	}
}

func TestActualizar_estadoInvalidoDevuelveConflicto(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	creado, _ := svc.Crear(ctx, paramsBase())
	if _, err := svc.ActualizarEstado(ctx, creado.ID, domain.Status("volado")); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Errorf("se esperaba ErrInvalidTransition, dio %v", err)
	}
}

func TestActualizar_tieneIdInexistente(t *testing.T) {
	svc := nuevoSvc()
	_, err := svc.Actualizar(context.Background(), "no-existe", paramsBase())
	if !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("se esperaba ErrNotFound, dio %v", err)
	}
}

func TestEliminar(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	creado, _ := svc.Crear(ctx, paramsBase())
	if err := svc.Eliminar(ctx, creado.ID); err != nil {
		t.Fatalf("Eliminar() error: %v", err)
	}
	if _, err := svc.Obtener(ctx, creado.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Error("el vehiculo deberia estar borrado")
	}
	if err := svc.Eliminar(ctx, creado.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Error("borrar dos veces deberia fallar")
	}
}

func TestListar_filtraYPagina(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	lote := []domain.Params{
		{Brand: "Toyota", Model: "Corolla", Year: 2018, PriceCents: 1_000_000},
		{Brand: "Toyota", Model: "RAV4", Year: 2022, PriceCents: 3_000_000},
		{Brand: "Honda", Model: "Civic", Year: 2020, PriceCents: 2_000_000},
	}
	for _, p := range lote {
		if _, err := svc.Crear(ctx, p); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("filtra por marca", func(t *testing.T) {
		res, err := svc.Listar(ctx, domain.Filtro{Brand: "Toyota"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Total != 2 {
			t.Errorf("se esperaban 2 Toyotas, hay %d", res.Total)
		}
	})

	t.Run("filtra por rango de precio", func(t *testing.T) {
		res, _ := svc.Listar(ctx, domain.Filtro{MinPrice: 1_500_000, MaxPrice: 2_500_000})
		if res.Total != 1 {
			t.Errorf("se esperaba 1 vehiculo en el rango, hay %d", res.Total)
		}
	})

	t.Run("busca texto libre", func(t *testing.T) {
		res, _ := svc.Listar(ctx, domain.Filtro{Search: "rav4"})
		if res.Total != 1 {
			t.Errorf("la busqueda 'rav4' deberia traer 1, trajo %d", res.Total)
		}
	})

	t.Run("pagina", func(t *testing.T) {
		res, _ := svc.Listar(ctx, domain.Filtro{Page: 2, PerPage: 2, OrderBy: "brand", Asc: true})
		if res.Total != 3 {
			t.Errorf("el total no cambia al paginar: %d", res.Total)
		}
		if len(res.Items) != 1 {
			t.Errorf("la pagina 2 de 2 deberia traer 1 elemento, trajo %d", len(res.Items))
		}
	})

	t.Run("pagina fuera de rango devuelve vacio", func(t *testing.T) {
		res, _ := svc.Listar(ctx, domain.Filtro{Page: 99, PerPage: 10})
		if len(res.Items) != 0 {
			t.Errorf("se esperaba lista vacia, trajo %d", len(res.Items))
		}
		if res.Total != 3 {
			t.Errorf("el total debe seguir reportandose: %d", res.Total)
		}
	})

	t.Run("per_page se acota a 100", func(t *testing.T) {
		res, _ := svc.Listar(ctx, domain.Filtro{PerPage: 5000})
		if res.PerPage != 100 {
			t.Errorf("se esperaba per_page acotado a 100, dio %d", res.PerPage)
		}
	})
}

// El vendedor puede corregir fichas, y PUT acepta un campo status. Este test
// fija que esa via NO esquiva la maquina de estados: reponer un vehiculo
// vendido a disponible sigue siendo una transicion prohibida.
func TestActualizar_noEsquivaLaMaquinaDeEstados(t *testing.T) {
	svc := nuevoSvc()
	ctx := context.Background()

	creado, _ := svc.Crear(ctx, paramsBase())
	if _, err := svc.ActualizarEstado(ctx, creado.ID, domain.StatusSold); err != nil {
		t.Fatal(err)
	}

	p := paramsBase()
	p.Model = "Corolla 2.0"
	p.Status = domain.StatusAvailable
	if _, err := svc.Actualizar(ctx, creado.ID, p); !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("PUT no debe poder deshacer una venta, dio %v", err)
	}

	// Y el vehiculo sigue vendido: el intento fallido no muta nada.
	tras, err := svc.Obtener(ctx, creado.ID)
	if err != nil {
		t.Fatal(err)
	}
	if tras.Status != domain.StatusSold {
		t.Errorf("el estado cambio a %q tras un intento fallido", tras.Status)
	}
	if tras.Model != "Corolla" {
		t.Errorf("el modelo cambio a %q tras un intento fallido", tras.Model)
	}
}
