package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"autoventa/backend/internal/modules/session/domain"
	"autoventa/backend/internal/modules/session/usecase"
	"autoventa/backend/internal/repos"
)

// fixture arma el servicio con repos en memoria y el hasher real de bcrypt,
// porque el comportamiento que interesa probar (que el hash no sea el password)
// depende del algoritmo.
type fixture struct {
	svc      *usecase.Service
	users    *repos.UserMemory
	sessions *repos.SessionMemory
	ahora    *time.Time
}

func nuevoFixture(t *testing.T, ttl time.Duration) *fixture {
	t.Helper()
	f := &fixture{
		users:    repos.NewUserMemory(),
		sessions: repos.NewSessionMemory(),
		ahora:    ptr(time.Now().UTC()),
	}
	f.svc = usecase.NewService(f.users, f.sessions, hasherReal(), ttl)
	return f
}

func TestCrearUsuario_noGuardaElPasswordEnClaro(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	u, err := f.svc.CrearUsuario(context.Background(), "Dueno@AutoVenta.com", "Dueño", "Secreto12345", domain.RoleDueño)
	if err != nil {
		t.Fatal(err)
	}
	if u.PasswordHash == "Secreto12345" {
		t.Fatal("el password se guardo en claro")
	}
	if u.PasswordHash == "" {
		t.Fatal("no se genero hash")
	}
	if u.Email != "dueno@autoventa.com" {
		t.Errorf("el email no se normalizo: %q", u.Email)
	}
	if !u.Activo {
		t.Error("un usuario nuevo debe quedar activo")
	}
}

func TestCrearUsuario_rechazaPasswordDebil(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	for _, pwd := range []string{"corta1A", "sinmayusculas123", "SinDigitosAqui"} {
		if _, err := f.svc.CrearUsuario(context.Background(), "a@b.com", "X", pwd, domain.RoleVendedor); !errors.Is(err, domain.ErrPasswordDebil) {
			t.Errorf("password %q: se esperaba ErrPasswordDebil, dio %v", pwd, err)
		}
	}
}

func TestCrearUsuario_rechazaEmailDuplicado(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	if _, err := f.svc.CrearUsuario(ctx, "dup@x.com", "A", "Secreto12345", domain.RoleVendedor); err != nil {
		t.Fatal(err)
	}
	// Distinto casing, mismo usuario: por eso NormalizarEmail.
	if _, err := f.svc.CrearUsuario(ctx, "DUP@X.COM", "B", "Secreto12345", domain.RoleVendedor); !errors.Is(err, domain.ErrEmailDuplicado) {
		t.Errorf("se esperaba ErrEmailDuplicado, dio %v", err)
	}
}

func TestCrearUsuario_rechazaRolInvalido(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	if _, err := f.svc.CrearUsuario(context.Background(), "a@b.com", "X", "Secreto12345", domain.Role("admin")); !errors.Is(err, domain.ErrRolInvalido) {
		t.Errorf("se esperaba ErrRolInvalido, dio %v", err)
	}
}

func TestLogin_credencialesValidas(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	f.svc.CrearUsuario(ctx, "vendedor@x.com", "Ana", "Secreto12345", domain.RoleVendedor)

	user, token, expira, err := f.svc.Login(ctx, "VENDEDOR@X.COM", "Secreto12345")
	if err != nil {
		t.Fatalf("Login() error: %v", err)
	}
	if token == "" {
		t.Error("token vacio")
	}
	if user.Role != domain.RoleVendedor {
		t.Errorf("rol inesperado: %q", user.Role)
	}
	if expira.IsZero() {
		t.Error("no se informo la expiracion")
	}
}

// El mensaje de error no debe distinguir entre "email no existe" y "password
// incorrecto": si lo hiciera, permitiria enumerar que emails estan registrados.
func TestLogin_noEnumeraUsuarios(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	f.svc.CrearUsuario(ctx, "existe@x.com", "Ana", "Secreto12345", domain.RoleVendedor)

	_, _, _, errInexistente := f.svc.Login(ctx, "nadie@x.com", "Secreto12345")
	_, _, _, errMalo := f.svc.Login(ctx, "existe@x.com", "PasswordMalo999")

	if !errors.Is(errInexistente, domain.ErrCredencialesInvalidas) {
		t.Errorf("email inexistente: se esperaba ErrCredencialesInvalidas, dio %v", errInexistente)
	}
	if !errors.Is(errMalo, domain.ErrCredencialesInvalidas) {
		t.Errorf("password malo: se esperaba ErrCredencialesInvalidas, dio %v", errMalo)
	}
	if errInexistente.Error() != errMalo.Error() {
		t.Errorf("los mensajes difieren y permiten enumerar: %q vs %q", errInexistente, errMalo)
	}
}

func TestLogin_usuarioInactivo(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	f.svc.CrearUsuario(ctx, "vendedor@x.com", "Ana", "Secreto12345", domain.RoleVendedor)

	usuarios, _ := f.svc.ListarUsuarios(ctx)
	if err := f.svc.Desactivar(ctx, domain.User{ID: "dueño-distinto"}, usuarios[0].ID); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := f.svc.Login(ctx, "vendedor@x.com", "Secreto12345"); !errors.Is(err, domain.ErrCredencialesInvalidas) {
		t.Errorf("un usuario desactivado no debe poder entrar, dio %v", err)
	}
}

func TestResolverAutenticado(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	creado, _ := f.svc.CrearUsuario(ctx, "dueno@x.com", "Dueño", "Secreto12345", domain.RoleDueño)
	_, token, _, _ := f.svc.Login(ctx, "dueno@x.com", "Secreto12345")

	resuelto, err := f.svc.ResolverAutenticado(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if resuelto.ID != creado.ID {
		t.Errorf("se resolvio otro usuario: %s", resuelto.ID)
	}
}

func TestResolverAutenticado_rechazaTokenInvalidoOCorto(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	for _, token := range []string{"", "basura", "a.b.c"} {
		if _, err := f.svc.ResolverAutenticado(ctx, token); !errors.Is(err, domain.ErrNoAutorizado) {
			t.Errorf("token %q: se esperaba ErrNoAutorizado, dio %v", token, err)
		}
	}
}

// Al expirar, el token guardado ya no sirve aunque el repositorio lo encuentre.
func TestResolverAutenticado_rechazaSesionVencida(t *testing.T) {
	f := nuevoFixture(t, -time.Hour) // TTL negativo forza una sesion ya vencida
	ctx := context.Background()
	f.svc.CrearUsuario(ctx, "dueno@x.com", "Dueño", "Secreto12345", domain.RoleDueño)
	// TTL negativo se normaliza al default, asi que expiramos a mano.
	_, token, _, _ := f.svc.Login(ctx, "dueno@x.com", "Secreto12345")
	vencida, _, err := domain.NuevaSession("x", time.Millisecond, time.Now().Add(-2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	_ = f.sessions.Save(ctx, SessionConHash(vencida, token))
	if _, err := f.svc.ResolverAutenticado(ctx, token); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Errorf("una sesion vencida no debe resolver, dio %v", err)
	}
}

func TestLogout_invalidaElToken(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	f.svc.CrearUsuario(ctx, "dueno@x.com", "Dueño", "Secreto12345", domain.RoleDueño)
	_, token, _, _ := f.svc.Login(ctx, "dueno@x.com", "Secreto12345")

	if _, err := f.svc.ResolverAutenticado(ctx, token); err != nil {
		t.Fatalf("el token deberia funcionar antes del logout: %v", err)
	}
	if err := f.svc.Logout(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ResolverAutenticado(ctx, token); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Error("tras el logout el token no debe resolver")
	}
}

func TestDesactivar_cierraLasSesionesAbiertas(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	dueno, _ := f.svc.CrearUsuario(ctx, "dueno@x.com", "Dueño", "Secreto12345", domain.RoleDueño)
	vendedor, _ := f.svc.CrearUsuario(ctx, "vendedor@x.com", "Ana", "Secreto12345", domain.RoleVendedor)
	_, token, _, _ := f.svc.Login(ctx, "vendedor@x.com", "Secreto12345")

	if err := f.svc.Desactivar(ctx, dueno, vendedor.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.ResolverAutenticado(ctx, token); !errors.Is(err, domain.ErrNoAutorizado) {
		t.Error("desactivar deberia cerrar las sesiones abiertas del usuario")
	}
}

// El dueño no puede quitarse a si mismo el acceso: si se degrada y es el unico
// administrador, el sistema queda sin administracion y nadie puede revertirlo.
func TestElDueñoNoPuedeDegradarseNiDesactivarse(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	dueno, _ := f.svc.CrearUsuario(ctx, "dueno@x.com", "Dueño", "Secreto12345", domain.RoleDueño)

	if _, err := f.svc.CambiarRol(ctx, dueno, dueno.ID, domain.RoleVendedor); err == nil {
		t.Error("el dueño no deberia poder degradarse")
	}
	if err := f.svc.Desactivar(ctx, dueno, dueno.ID); err == nil {
		t.Error("el dueño no deberia poder desactivarse")
	}
}

func TestCambiarRol(t *testing.T) {
	f := nuevoFixture(t, time.Hour)
	ctx := context.Background()
	dueno, _ := f.svc.CrearUsuario(ctx, "dueno@x.com", "Dueño", "Secreto12345", domain.RoleDueño)
	vendedor, _ := f.svc.CrearUsuario(ctx, "vendedor@x.com", "Ana", "Secreto12345", domain.RoleVendedor)

	actualizado, err := f.svc.CambiarRol(ctx, dueno, vendedor.ID, domain.RoleDueño)
	if err != nil {
		t.Fatal(err)
	}
	if actualizado.Role != domain.RoleDueño {
		t.Errorf("el rol no cambio: %q", actualizado.Role)
	}
	if _, err := f.svc.CambiarRol(ctx, dueno, vendedor.ID, domain.Role("root")); !errors.Is(err, domain.ErrRolInvalido) {
		t.Errorf("se esperaba ErrRolInvalido, dio %v", err)
	}
}
