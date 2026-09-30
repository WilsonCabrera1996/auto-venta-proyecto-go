package domain

import "testing"

func TestMatrizDePermisos(t *testing.T) {
	casos := []struct {
		rol      Role
		permiso  Permiso
		esperado bool
	}{
		// El dueño administra todo.
		{RoleDueño, PermVehiculoVer, true},
		{RoleDueño, PermVehiculoCrear, true},
		{RoleDueño, PermVehiculoEditar, true},
		{RoleDueño, PermVehiculoEliminar, true},
		{RoleDueño, PermVehiculoCambiarEstado, true},
		{RoleDueño, PermReporteVentas, true},
		{RoleDueño, PermUsuarioAdministrar, true},

		// El vendedor opera: ve, carga, corrige y marca vendido. Nada mas.
		{RoleVendedor, PermVehiculoVer, true},
		{RoleVendedor, PermVehiculoCrear, true},
		{RoleVendedor, PermVehiculoEditar, true},
		{RoleVendedor, PermVehiculoCambiarEstado, true},
		{RoleVendedor, PermVehiculoEliminar, false},
		{RoleVendedor, PermReporteVentas, false},
		{RoleVendedor, PermUsuarioAdministrar, false},

		// Un rol desconocido no concede nada.
		{Role("admin"), PermVehiculoEliminar, false},
		{Role(""), PermVehiculoVer, false},
	}
	for _, tc := range casos {
		if got := tc.rol.Puede(tc.permiso); got != tc.esperado {
			t.Errorf("rol %q permiso %q = %v, se esperaba %v", tc.rol, tc.permiso, got, tc.esperado)
		}
	}
}

func TestRoleValid(t *testing.T) {
	if !RoleDueño.Valid() || !RoleVendedor.Valid() {
		t.Error("los roles definidos deben ser validos")
	}
	for _, r := range []Role{"", "admin", "DUENO", "dueño"} {
		if r.Valid() {
			t.Errorf("el rol %q no deberia ser valido", r)
		}
	}
}

func TestValidarPasswordStrength(t *testing.T) {
	casos := []struct {
		nombre   string
		password string
		queremos bool
	}{
		{"corta", "Ab1", false},
		{"sin mayuscula", "secreto12345", false},
		{"sin digito", "SecretoMuyLargo", false},
		{"larga demais", "Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234Abcd1234", false},
		{"correcta", "Secreto12345", true},
	}
	for _, tc := range casos {
		err := ValidarPasswordStrength(tc.password)
		if tc.queremos && err != nil {
			t.Errorf("%s: se esperaba valida, dio %v", tc.nombre, err)
		}
		if !tc.queremos && err == nil {
			t.Errorf("%s: se esperaba error", tc.nombre)
		}
	}
}

// El limite de 72 bytes no es cosmetico: bcrypt trunca en silencio, asi que un
// password de 100 chars produciria un hash que no representa lo que el usuario
// cree. Por eso el dominio lo rechaza en vez de dejarlo pasar.
func TestPasswordLargoSeRechaza(t *testing.T) {
	largo := "Abcd1234"
	for i := 0; i < 10; i++ {
		largo += "abcdefgh"
	}
	if err := ValidarPasswordStrength(largo); err == nil {
		t.Error("un password de mas de 72 bytes debe rechazarse")
	}
}

func TestNormalizarEmail(t *testing.T) {
	casos := map[string]string{
		"  Dueño@AutoVenta.COM ": "dueño@autoventa.com",
		"vendedor@x.com":         "vendedor@x.com",
	}
	for in, want := range casos {
		if got := NormalizarEmail(in); got != want {
			t.Errorf("NormalizarEmail(%q) = %q, se esperaba %q", in, got, want)
		}
	}
}
