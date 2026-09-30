package usecase_test

import (
	"time"

	"autoventa/backend/internal/infra/crypto"
	"autoventa/backend/internal/modules/session/domain"
)

func ptr[T any](v T) *T { return &v }

// hasherReal usa el bcrypt de produccion. Un fake trivial serviria para la
// logica, pero entonces los tests no detectarian si alguien guardara el
// password en claro.
func hasherReal() domain.Hasher { return crypto.NewBcryptHasher() }

// SessionConHash fuerza el hash del token para simular una sesion vencida sin
// tener que dormir.
func SessionConHash(s domain.Session, token string) domain.Session {
	s.TokenHash = domain.HashToken(token)
	s.Creada = time.Now().Add(-2 * time.Hour).UTC()
	s.Expira = time.Now().Add(-time.Hour).UTC()
	return s
}
