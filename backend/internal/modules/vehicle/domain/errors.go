package domain

import "errors"

// Errores de dominio. El adaptador HTTP los traduce a codigos HTTP; el dominio
// no conoce HTTP.
var (
	ErrNotFound          = errors.New("vehiculo no encontrado")
	ErrAlreadyExists     = errors.New("vehiculo ya existe")
	ErrValidation        = errors.New("datos invalidos")
	ErrInvalidTransition = errors.New("transicion de estado invalida")
)
