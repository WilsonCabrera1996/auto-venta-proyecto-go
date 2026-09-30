package usecase

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"autoventa/backend/internal/modules/session/domain"
	"autoventa/backend/internal/platform/id"
)

// Service orquesta los casos de uso de sesion. No sabe nada de HTTP: recibe
// credenciales o identificadores y devuelve resultados o errores de dominio.
type Service struct {
	users    domain.UserRepository
	sessions domain.SessionRepository
	hasher   domain.Hasher
	ttl      time.Duration
	ahora    func() time.Time
}

func NewService(users domain.UserRepository, sessions domain.SessionRepository, hasher domain.Hasher, ttl time.Duration) *Service {
	if ttl <= 0 {
		ttl = domain.TTLDefault
	}
	return &Service{
		users:    users,
		sessions: sessions,
		hasher:   hasher,
		ttl:      ttl,
		ahora:    func() time.Time { return time.Now().UTC() },
	}
}

// CrearUsuario registra una identidad. El password entra en claro por primera
// vez y solo se guarda su hash. El caller decide el rol; en HTTP lo restringe
// el permiso PermUsuarioAdministrar.
func (s *Service) CrearUsuario(ctx context.Context, email, nombre, password string, rol domain.Role) (domain.User, error) {
	if err := domain.ValidarPasswordStrength(password); err != nil {
		return domain.User{}, err
	}
	if !rol.Valid() {
		return domain.User{}, domain.ErrRolInvalido
	}
	norm := domain.NormalizarEmail(email)
	if norm == "" || !contieneArroba(norm) {
		return domain.User{}, fmt.Errorf("%w: email invalido", domain.ErrValidacion)
	}
	if _, err := s.users.GetByEmail(ctx, norm); err == nil {
		return domain.User{}, domain.ErrEmailDuplicado
	}
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return domain.User{}, err
	}
	ahora := s.ahora()
	return s.users.Create(ctx, domain.User{
		ID:           id.New(),
		Email:        norm,
		Nombre:       strings.TrimSpace(nombre),
		PasswordHash: hash,
		Role:         rol,
		Activo:       true,
		CreatedAt:    ahora,
		UpdatedAt:    ahora,
	})
}

// contieneArroba es un chequeo deliberadamente laxo: la validacion real del
// formato la hara el schema de Postgres. Aca solo se descarta lo obvio.
func contieneArroba(email string) bool {
	for i := 0; i < len(email); i++ {
		if email[i] == '@' && i > 0 && i < len(email)-1 {
			return true
		}
	}
	return false
}

// Login valida credenciales y emite sesion. Devuelve el token EN CLARO: es la
// unica vez que sale del proceso, despues solo se guarda su hash.
func (s *Service) Login(ctx context.Context, email, password string) (domain.User, string, time.Time, error) {
	u, err := s.users.GetByEmail(ctx, domain.NormalizarEmail(email))
	if err != nil {
		// Se devuelve siempre el mismo error para no revelar si el email existe.
		return domain.User{}, "", time.Time{}, domain.ErrCredencialesInvalidas
	}
	if !u.Activo {
		return domain.User{}, "", time.Time{}, domain.ErrCredencialesInvalidas
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		return domain.User{}, "", time.Time{}, domain.ErrCredencialesInvalidas
	}
	sess, token, err := domain.NuevaSession(u.ID, s.ttl, s.ahora())
	if err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	if err := s.sessions.Save(ctx, sess); err != nil {
		return domain.User{}, "", time.Time{}, err
	}
	return u, token, sess.Expira, nil
}

// ResolverAutenticado traduce un token en claro a la identidad que lo posee.
// Es lo que usa el middleware de autenticacion en cada request.
func (s *Service) ResolverAutenticado(ctx context.Context, token string) (domain.User, error) {
	if token == "" {
		return domain.User{}, domain.ErrNoAutorizado
	}
	sess, err := s.sessions.GetByTokenHash(ctx, domain.HashToken(token))
	if err != nil {
		return domain.User{}, domain.ErrNoAutorizado
	}
	if !sess.Activa(s.ahora()) {
		return domain.User{}, domain.ErrNoAutorizado
	}
	u, err := s.users.GetByID(ctx, sess.UserID)
	if err != nil {
		return domain.User{}, domain.ErrNoAutorizado
	}
	if !u.Activo {
		return domain.User{}, domain.ErrNoAutorizado
	}
	return u, nil
}

// Logout revoca la sesion actual.
func (s *Service) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.Revoke(ctx, domain.HashToken(token), s.ahora())
}

// Yo devuelve la identidad autenticada, para que el frontend sepa que permisos
// tiene y muestre la UI correcta sin adivinar.
func (s *Service) Yo(ctx context.Context, actor domain.User) (domain.User, error) {
	if actor.ID == "" {
		return domain.User{}, domain.ErrNoAutorizado
	}
	return s.users.GetByID(ctx, actor.ID)
}

// ListarUsuarios es caso de uso del dueño.
func (s *Service) ListarUsuarios(ctx context.Context) ([]domain.User, error) {
	return s.users.List(ctx)
}

// CambiarRol altera el rol de un usuario. Se prohibe tocar al propio dueño:
// si el dueño se degrada a vendedor y es el ultimo, el sistema queda sin
// administracion y no hay quien lo revierta.
func (s *Service) CambiarRol(ctx context.Context, actor domain.User, objetivoID string, nuevo domain.Role) (domain.User, error) {
	if !nuevo.Valid() {
		return domain.User{}, domain.ErrRolInvalido
	}
	if actor.ID == objetivoID {
		return domain.User{}, errors.New("el dueño no puede cambiar su propio rol")
	}
	objetivo, err := s.users.GetByID(ctx, objetivoID)
	if err != nil {
		return domain.User{}, err
	}
	objetivo.Role = nuevo
	objetivo.UpdatedAt = s.ahora()
	return s.users.Update(ctx, objetivo)
}

// Desactivar da de baja a un usuario y cierra todas sus sesiones abiertas.
func (s *Service) Desactivar(ctx context.Context, actor domain.User, objetivoID string) error {
	if actor.ID == objetivoID {
		return errors.New("el dueño no puede desactivar su propia cuenta")
	}
	objetivo, err := s.users.GetByID(ctx, objetivoID)
	if err != nil {
		return err
	}
	objetivo.Activo = false
	objetivo.UpdatedAt = s.ahora()
	if _, err := s.users.Update(ctx, objetivo); err != nil {
		return err
	}
	return s.sessions.RevokeAllForUser(ctx, objetivo.ID, s.ahora())
}
