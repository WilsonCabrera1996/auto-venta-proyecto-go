package domain

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func paramsValidas() Params {
	return Params{
		Brand:      "Toyota",
		Model:      "Corolla",
		Year:       2020,
		PriceCents: 1_850_000,
	}
}

func TestNew_normalizaYCumpleInvariantes(t *testing.T) {
	v, err := New(Params{
		Brand:      "  Toyota  ",
		Model:      "  Corolla  ",
		Year:       2021,
		PriceCents: 2_000_000,
		VIN:        " 1hgcm82633a004352 ",
		Drivetrain: "fwd",
		FuelType:   "GASOLINA",
	})
	if err != nil {
		t.Fatalf("New() error inesperado: %v", err)
	}
	if v.Brand != "Toyota" {
		t.Errorf("marca no normalizada: %q", v.Brand)
	}
	if v.Model != "Corolla" {
		t.Errorf("modelo no normalizado: %q", v.Model)
	}
	if v.VIN != "1HGCM82633A004352" {
		t.Errorf("VIN no normalizado: %q", v.VIN)
	}
	if v.Drivetrain != "FWD" {
		t.Errorf("drivetrain no normalizado: %q", v.Drivetrain)
	}
	if v.FuelType != "gasolina" {
		t.Errorf("fuel type no normalizado: %q", v.FuelType)
	}
	if v.Status != StatusAvailable {
		t.Errorf("estado por defecto deberia ser available, dio %q", v.Status)
	}
	if v.ID == "" {
		t.Error("el dominio debe asignar el ID")
	}
	if v.CreatedAt.IsZero() || v.UpdatedAt.IsZero() {
		t.Error("timestamps no fijados por el dominio")
	}
}

func TestNew_noCambiaElCasingDeMarcaYModelo(t *testing.T) {
	// A proposito: el dominio recorta espacios pero no reescribe el casing.
	// "BMW" y "KIA" deben sobrevivir intactas, y el title-casing automatico
	// destrozaria marcas compuestas como "alfa romeo". El casing es cosa
	// de presentacion, no de negocio.
	v, err := New(Params{Brand: "BMW", Model: "i4", Year: 2022, PriceCents: 1})
	if err != nil {
		t.Fatal(err)
	}
	if v.Brand != "BMW" {
		t.Errorf("la marca debe conservarse tal cual, dio %q", v.Brand)
	}
	if v.Model != "i4" {
		t.Errorf("el modelo debe conservarse tal cual, dio %q", v.Model)
	}
}

func TestNew_rechazaInvariantesViolados(t *testing.T) {
	casos := []struct {
		nombre string
		mutar  func(*Params)
	}{
		{"marca vacia", func(p *Params) { p.Brand = "" }},
		{"modelo vacio", func(p *Params) { p.Model = "" }},
		{"anio viejo", func(p *Params) { p.Year = 1800 }},
		{"anio futuro", func(p *Params) { p.Year = time.Now().Year() + 5 }},
		{"precio cero", func(p *Params) { p.PriceCents = 0 }},
		{"precio negativo", func(p *Params) { p.PriceCents = -1 }},
		{"kilometraje negativo", func(p *Params) { p.Mileage = -1 }},
		{"estado invalido", func(p *Params) { p.Status = "volado" }},
		{"VIN corto", func(p *Params) { p.VIN = "123" }},
		{"VIN con I", func(p *Params) { p.VIN = "8AJ11G704I234567" }},
		{"VIN con O", func(p *Params) { p.VIN = "8AJ11G7O41234567" }},
		{"VIN con Q", func(p *Params) { p.VIN = "8AJ11G704Q234567" }},
		{"demasiadas puertas", func(p *Params) { p.Doors = 7 }},
		{"demasiados asientos", func(p *Params) { p.Seats = 10 }},
		{"cilindrada negativa", func(p *Params) { p.EngineCC = -1 }},
		{"potencia negativa", func(p *Params) { p.Horsepower = -1 }},
	}
	for _, tc := range casos {
		t.Run(tc.nombre, func(t *testing.T) {
			p := paramsValidas()
			tc.mutar(&p)
			_, err := New(p)
			if err == nil {
				t.Fatal("se esperaba error de validacion")
			}
			if !errors.Is(err, ErrValidation) {
				t.Errorf("se esperaba ErrValidation, dio %v", err)
			}
		})
	}
}

func TestNew_agregaTodosLosProblemasDeUnaVez(t *testing.T) {
	_, err := New(Params{})
	if err == nil {
		t.Fatal("se esperaba error")
	}
	// Marca, modelo, anio y precio fallan a la vez; el mensaje debe listarlos.
	for _, quiero := range []string{"marca", "modelo", "anio", "precio"} {
		if !strings.Contains(err.Error(), quiero) {
			t.Errorf("mensaje de error incompleto, falta %q: %v", quiero, err)
		}
	}
}

func TestTransicionDeEstados(t *testing.T) {
	v, err := New(paramsValidas())
	if err != nil {
		t.Fatal(err)
	}

	if err := v.TransitionTo(StatusReserved); err != nil {
		t.Errorf("available -> reserved deberia ser valida: %v", err)
	}
	if v.Status != StatusReserved {
		t.Errorf("estado no actualizado: %q", v.Status)
	}
	if err := v.TransitionTo(StatusSold); err != nil {
		t.Errorf("reserved -> sold deberia ser valida: %v", err)
	}
	if v.Disponible() {
		t.Error("un vehiculo vendido no debe estar disponible")
	}

	// sold -> available esta restringido: es el unico caso reingreso y exige
	// pasar por el caso de uso, no un update cualquiera.
	if err := v.TransitionTo(StatusAvailable); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("sold -> available deberia fallar, dio %v", err)
	}
	if v.Status != StatusSold {
		t.Error("un intento fallido no debe mutar el estado")
	}
}

func TestTransicionAlMismoEstadoEsNoOp(t *testing.T) {
	v, _ := New(paramsValidas())
	if err := v.TransitionTo(StatusAvailable); err != nil {
		t.Errorf("transicion al mismo estado deberia ser no-op, dio %v", err)
	}
}

func TestTransicionAEstadoInvalido(t *testing.T) {
	v, _ := New(paramsValidas())
	if err := v.TransitionTo(Status("hipotecado")); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("se esperaba ErrInvalidTransition, dio %v", err)
	}
}
