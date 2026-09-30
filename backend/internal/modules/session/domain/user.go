package domain

import (
	"fmt"
	"strings"
	"time"
)

// Role es el rol del usuario. Es un tipo propio, no string, para que un rol
// inventado no se pueda colar por un endpoint.
type Role string

const (
	// RoleDueño administra el inventario y ve la conciliacion de ventas.
	RoleDueño Role = "dueno"
	// RoleVendedor carga vehiculos y los marca como vendidos. Nada mas.
	RoleVendedor Role = "vendedor"
)

func (r Role) Valid() bool {
	switch r {
	case RoleDueño, RoleVendedor:
		return true
	}
	return false
}

// Permiso es una capacidad concreta. Los guards del router verifican permisos,
// no roles: asi el dominio define que puede hacer cada rol y el middleware
// queda como una simple comprobacion.
type Permiso string

const (
	PermVehiculoVer           Permiso = "vehiculo:ver"
	PermVehiculoCrear         Permiso = "vehiculo:crear"
	PermVehiculoEditar        Permiso = "vehiculo:editar"
	PermVehiculoEliminar      Permiso = "vehiculo:eliminar"
	PermVehiculoCambiarEstado Permiso = "vehiculo:cambiar_estado"
	PermUsuarioAdministrar    Permiso = "usuario:administrar"
	PermReporteVentas         Permiso = "reporte:ventas"
)

// permisosPorRol es la unica fuente de verdad de la matriz de authorization.
// El dueño tiene todo; el vendedor solo lo que necesita para operar el dia a dia.
var permisosPorRol = map[Role][]Permiso{
	RoleDueño: {
		PermVehiculoVer,
		PermVehiculoCrear,
		PermVehiculoEditar,
		PermVehiculoEliminar,
		PermVehiculoCambiarEstado,
		PermUsuarioAdministrar,
		PermReporteVentas,
	},
	RoleVendedor: {
		PermVehiculoVer,
		PermVehiculoCrear,
		// El vendedor corrige fichas (un precio mal cargado, una patente) pero
		// no borra ni administra. Un cambio de estado tambien se puede hacer por
		// esta via si viene en el cuerpo; sigue pasando por la maquina de
		// estados, asi que no abre una puerta extra.
		PermVehiculoEditar,
		PermVehiculoCambiarEstado,
	},
}

// PermisosDe devuelve la lista de permisos del rol. Se usa para el DTO de
// usuario, para que el frontend pueda decidir que mostrar sin duplicar la
// matriz de permisos en el cliente.
func PermisosDe(r Role) []Permiso {
	return permisosPorRol[r]
}

// Puede indica si el rol otorga el permiso. Dominio puro, sin HTTP.
func (r Role) Puede(p Permiso) bool {
	for _, h := range permisosPorRol[r] {
		if h == p {
			return true
		}
	}
	return false
}

// User es la identidad. El password nunca viaja en claro: solo su hash bcrypt.
type User struct {
	ID           string
	Email        string
	Nombre       string
	PasswordHash string
	Role         Role
	Activo       bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

const (
	passwordMinimo = 12
	passwordMaximo = 72 // bcrypt trunca silenciosamente a 72 bytes
)

// NormalizarEmail deja el email en la forma canonica con la que se indexa el
// UNIQUE. Sin esto, "Dueno@X.com" y "dueno@x.com" serian dos usuarios.
func NormalizarEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

// ValidarPasswordStrength es una invariante de dominio: la longitud minima
// vive aca, no en el handler ni en el cliente.
func ValidarPasswordStrength(password string) error {
	if len(password) < passwordMinimo {
		return fmt.Errorf("%w: minimo %d caracteres", ErrPasswordDebil, passwordMinimo)
	}
	if len(password) > passwordMaximo {
		// bcrypt no lee mas alla de 72 bytes; sin esto un password largo
		// produciria un hash que no corresponde a lo que el usuario cree.
		return fmt.Errorf("%w: maximo %d caracteres", ErrPasswordDebil, passwordMaximo)
	}
	var tieneMayuscula, tieneDigito bool
	for _, c := range password {
		switch {
		case c >= 'A' && c <= 'Z':
			tieneMayuscula = true
		case c >= '0' && c <= '9':
			tieneDigito = true
		}
	}
	if !tieneMayuscula || !tieneDigito {
		return fmt.Errorf("%w: requiere al menos una mayuscula y un digito", ErrPasswordDebil)
	}
	return nil
}
