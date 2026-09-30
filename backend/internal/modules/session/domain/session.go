package domain

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"autoventa/backend/internal/platform/id"
)

// TTL por defecto de una sesion. Corto a proposito: el sistema es de uso
// interno con un handful de usuarios, no un bank. Si alguien deja la sesion
// abierta en una maquina compartida, expira sola.
const TTLDefault = 8 * time.Hour

// Session es una sesion emitida. Solo se persiste el TokenHash: el token en
// claro existe unicamente en la respuesta HTTP al usuario que se autentico.
type Session struct {
	ID        string
	UserID    string
	TokenHash string
	Creada    time.Time
	Expira    time.Time
	Revocada  *time.Time
}

// Activa indica si la sesion sirve ahora mismo.
func (s Session) Activa(ahora time.Time) bool {
	if s.Revocada != nil {
		return false
	}
	return ahora.Before(s.Expira)
}

// NuevaSession emite una sesion para el usuario y devuelve el token en claro.
// El token tiene 256 bits de entropia: no es adivinable ni por fuerza bruta.
func NuevaSession(userID string, ttl time.Duration, ahora time.Time) (Session, string, error) {
	if userID == "" {
		return Session{}, "", fmt.Errorf("%w: userID vacio", ErrSesionInvalida)
	}
	if ttl <= 0 {
		ttl = TTLDefault
	}
	token, err := generarToken()
	if err != nil {
		return Session{}, "", err
	}
	return Session{
		ID:        id.New(),
		UserID:    userID,
		TokenHash: HashToken(token),
		Creada:    ahora,
		Expira:    ahora.Add(ttl),
	}, token, nil
}

func generarToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("no se pudo generar token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:]), nil
}

// HashToken es la funcion de lookup del repositorio. SHA-256 alcanza porque el
// token es aleatorio de 256 bits: no hay diccionario que atacar. Si alguna vez
// el token fuera elegido por el usuario, esto habria que cambiar por bcrypt/argon2.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
