package domain

import "errors"

// Errores de dominio del modulo de sesiones. respond los traduce a HTTP:
// credenciales malas -> 401, sin permiso -> 403.
var (
	ErrCredencialesInvalidas = errors.New("credenciales invalidas")
	ErrNoAutorizado          = errors.New("no autenticado")
	ErrProhibido             = errors.New("sin permisos")
	ErrEmailDuplicado        = errors.New("email ya registrado")
	ErrPasswordDebil         = errors.New("password insegura")
	ErrRolInvalido           = errors.New("rol invalido")
	ErrValidacion            = errors.New("datos invalidos")
	ErrUsuarioNoEncontrado   = errors.New("usuario no encontrado")
	ErrSesionInvalida        = errors.New("sesion invalida o expirada")
)
