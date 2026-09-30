package domain

import (
	"testing"
	"time"
)

func TestNuevaSession_noAlmacenaElTokenEnClaro(t *testing.T) {
	ahora := time.Now().UTC()
	s, token, err := NuevaSession("user-1", time.Hour, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("token vacio")
	}
	if s.TokenHash == token {
		t.Error("el token en claro no debe quedar persistido")
	}
	if s.TokenHash != HashToken(token) {
		t.Error("TokenHash debe ser el sha256 del token")
	}
	if len(token) < 40 {
		t.Errorf("token demasiado corto para ser aleatorio: %d", len(token))
	}
}

func TestNuevaSession_tokensUnicos(t *testing.T) {
	ahora := time.Now().UTC()
	vistos := make(map[string]bool)
	for i := 0; i < 500; i++ {
		_, token, _ := NuevaSession("user-1", time.Hour, ahora)
		if vistos[token] {
			t.Fatal("se genero un token repetido")
		}
		vistos[token] = true
	}
}

func TestSessionActiva(t *testing.T) {
	ahora := time.Now().UTC()

	vigente, _, _ := NuevaSession("u", time.Hour, ahora)
	if !vigente.Activa(ahora) {
		t.Error("una sesion recien creada debe estar activa")
	}
	if !vigente.Activa(ahora.Add(59 * time.Minute)) {
		t.Error("debe seguir activa antes de expirar")
	}
	if vigente.Activa(ahora.Add(2 * time.Hour)) {
		t.Error("una sesion vencida no debe estar activa")
	}

	revocada := vigente
	revocadaEn := ahora
	revocada.Revocada = &revocadaEn
	if revocada.Activa(ahora) {
		t.Error("una sesion revocada no debe estar activa ni al instante de revocarla")
	}
}

func TestNuevaSession_ttlPorDefecto(t *testing.T) {
	ahora := time.Now().UTC()
	s, _, err := NuevaSession("u", 0, ahora)
	if err != nil {
		t.Fatal(err)
	}
	if !s.Expira.Equal(ahora.Add(TTLDefault)) {
		t.Errorf("TTL por defecto no aplicado: %v", s.Expira.Sub(ahora))
	}
}

func TestNuevaSession_rechazaUserIDVacio(t *testing.T) {
	if _, _, err := NuevaSession("", time.Hour, time.Now()); err == nil {
		t.Error("debe rechazar userID vacio")
	}
}
