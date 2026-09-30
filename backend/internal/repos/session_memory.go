package repos

import (
	"context"
	"sort"
	"sync"
	"time"

	"autoventa/backend/internal/modules/session/domain"
)

// UserMemory guarda identidades en memoria. El email es la clave secundaria:
// se indexa aparte para no recorrer el mapa en cada login.
type UserMemory struct {
	mu      sync.RWMutex
	byID    map[string]domain.User
	byEmail map[string]string
}

func NewUserMemory() *UserMemory {
	return &UserMemory{
		byID:    make(map[string]domain.User),
		byEmail: make(map[string]string),
	}
}

func (r *UserMemory) Create(_ context.Context, u domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, existe := r.byID[u.ID]; existe {
		return domain.User{}, domain.ErrEmailDuplicado
	}
	if _, existe := r.byEmail[u.Email]; existe {
		return domain.User{}, domain.ErrEmailDuplicado
	}
	r.byID[u.ID] = u
	r.byEmail[u.Email] = u.ID
	return u, nil
}

func (r *UserMemory) GetByID(_ context.Context, id string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.byID[id]
	if !ok {
		return domain.User{}, domain.ErrUsuarioNoEncontrado
	}
	return u, nil
}

func (r *UserMemory) GetByEmail(_ context.Context, email string) (domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEmail[email]
	if !ok {
		return domain.User{}, domain.ErrUsuarioNoEncontrado
	}
	return r.byID[id], nil
}

func (r *UserMemory) List(_ context.Context) ([]domain.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]domain.User, 0, len(r.byID))
	for _, u := range r.byID {
		out = append(out, u)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt.Before(out[j].CreatedAt) })
	return out, nil
}

func (r *UserMemory) Update(_ context.Context, u domain.User) (domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	previo, ok := r.byID[u.ID]
	if !ok {
		return domain.User{}, domain.ErrUsuarioNoEncontrado
	}
	if previo.Email != u.Email {
		delete(r.byEmail, previo.Email)
		r.byEmail[u.Email] = u.ID
	}
	r.byID[u.ID] = u
	return u, nil
}

// SessionMemory guarda sesiones indexadas por TokenHash. Nunca por el token en
// claro, que es lo unico que el repositorio debe poder ver.
type SessionMemory struct {
	mu   sync.RWMutex
	byID map[string]domain.Session // por tokenHash
}

func NewSessionMemory() *SessionMemory {
	return &SessionMemory{byID: make(map[string]domain.Session)}
}

func (r *SessionMemory) Save(_ context.Context, s domain.Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.byID[s.TokenHash] = s
	return nil
}

func (r *SessionMemory) GetByTokenHash(_ context.Context, tokenHash string) (domain.Session, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	s, ok := r.byID[tokenHash]
	if !ok {
		return domain.Session{}, domain.ErrSesionInvalida
	}
	return s, nil
}

func (r *SessionMemory) Revoke(_ context.Context, tokenHash string, ahora time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s, ok := r.byID[tokenHash]
	if !ok {
		return domain.ErrSesionInvalida
	}
	t := ahora
	s.Revocada = &t
	r.byID[tokenHash] = s
	return nil
}

func (r *SessionMemory) RevokeAllForUser(_ context.Context, userID string, ahora time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for hash, s := range r.byID {
		if s.UserID == userID && s.Revocada == nil {
			t := ahora
			s.Revocada = &t
			r.byID[hash] = s
		}
	}
	return nil
}
