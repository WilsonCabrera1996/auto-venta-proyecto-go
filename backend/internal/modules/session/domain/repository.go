package domain

import (
	"context"
	"time"
)

// UserRepository es el puerto de salida de identidades.
type UserRepository interface {
	Create(ctx context.Context, u User) (User, error)
	GetByID(ctx context.Context, id string) (User, error)
	GetByEmail(ctx context.Context, email string) (User, error)
	List(ctx context.Context) ([]User, error)
	Update(ctx context.Context, u User) (User, error)
}

// SessionRepository es el puerto de salida de sesiones. Todas las operaciones
// reciben el token ya hasheado: el repositorio nunca ve el token en claro.
type SessionRepository interface {
	Save(ctx context.Context, s Session) error
	GetByTokenHash(ctx context.Context, tokenHash string) (Session, error)
	Revoke(ctx context.Context, tokenHash string, ahora time.Time) error
	RevokeAllForUser(ctx context.Context, userID string, ahora time.Time) error
}

// Hasher abstrae el algoritmo de hashing de passwords. Vive en el dominio como
// puerto y lo implementa infra, para que el dominio no dependa de bcrypt.
type Hasher interface {
	Hash(password string) (string, error)
	Compare(hash, password string) error
}
