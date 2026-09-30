// Package respond centraliza la forma de las respuestas HTTP. Todos los handlers
// pasan por aqui, para que el contrato de error sea uniforme en todos los modulos.
package respond

import (
	"encoding/json"
	"errors"
	"log/slog"
	"mime"
	"net/http"

	"autoventa/backend/internal/modules/session/domain"
	vehicledomain "autoventa/backend/internal/modules/vehicle/domain"
)

// Error es el cuerpo estandar de error.
type Error struct {
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// Errores propios de la capa de transporte. No viven en el dominio porque el
// dominio no debe saber de HTTP.
type errorHTTP struct {
	msg    string
	status int
	code   string
}

func (e errorHTTP) Error() string { return e.msg }

// BadRequest construye un 400 con mensaje propio.
func BadRequest(msg string) error {
	return errorHTTP{msg: msg, status: http.StatusBadRequest, code: "bad_request"}
}

// ErrInvalidJSON se usa cuando el cuerpo no es decodificable.
var ErrInvalidJSON = errors.New("json invalido")

// Fail traduce errores de dominio a codigos HTTP. Los errores que ningun modulo
// conoce se devuelven como 500 sin filtrar detalle al cliente.
func Fail(w http.ResponseWriter, err error) {
	var httpErr errorHTTP
	if errors.As(err, &httpErr) {
		write(w, httpErr.status, Error{Message: httpErr.msg, Code: httpErr.code})
		return
	}

	switch {
	// --- sesion: 401 / 403 ---
	case errors.Is(err, domain.ErrCredencialesInvalidas):
		unauthorized(w, "credenciales invalidas", "invalid_credentials")
	case errors.Is(err, domain.ErrNoAutorizado), errors.Is(err, domain.ErrSesionInvalida):
		unauthorized(w, "no autenticado", "unauthorized")
	case errors.Is(err, domain.ErrProhibido):
		write(w, http.StatusForbidden, Error{Message: "sin permisos", Code: "forbidden"})
	case errors.Is(err, domain.ErrEmailDuplicado):
		write(w, http.StatusConflict, Error{Message: err.Error(), Code: "conflict"})
	case errors.Is(err, domain.ErrUsuarioNoEncontrado):
		write(w, http.StatusNotFound, Error{Message: err.Error(), Code: "not_found"})
	case errors.Is(err, domain.ErrPasswordDebil),
		errors.Is(err, domain.ErrRolInvalido),
		errors.Is(err, domain.ErrValidacion):
		write(w, http.StatusUnprocessableEntity, Error{Message: err.Error(), Code: "validation_error"})

	// --- vehiculo ---
	case errors.Is(err, vehicledomain.ErrNotFound):
		write(w, http.StatusNotFound, Error{Message: err.Error(), Code: "not_found"})
	case errors.Is(err, vehicledomain.ErrAlreadyExists):
		write(w, http.StatusConflict, Error{Message: err.Error(), Code: "conflict"})
	case errors.Is(err, vehicledomain.ErrValidation):
		write(w, http.StatusUnprocessableEntity, Error{Message: err.Error(), Code: "validation_error"})
	case errors.Is(err, vehicledomain.ErrInvalidTransition):
		write(w, http.StatusConflict, Error{Message: err.Error(), Code: "invalid_transition"})

	// --- transporte ---
	case errors.Is(err, ErrInvalidJSON):
		write(w, http.StatusBadRequest, Error{Message: "cuerpo JSON invalido", Code: "bad_request"})

	default:
		slog.Error("error no controlado", "err", err)
		write(w, http.StatusInternalServerError, Error{Message: "error interno", Code: "internal"})
	}
}

func unauthorized(w http.ResponseWriter, msg, code string) {
	// WWW-Authenticate es lo que un cliente HTTP espera para saber que debe
	// mandar un Bearer token.
	w.Header().Set("WWW-Authenticate", `Bearer realm="autoventa"`)
	write(w, http.StatusUnauthorized, Error{Message: msg, Code: code})
}

func OK(w http.ResponseWriter, data any) {
	write(w, http.StatusOK, data)
}

func Created(w http.ResponseWriter, data any) {
	write(w, http.StatusCreated, data)
}

func write(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if body == nil {
		return
	}
	_ = json.NewEncoder(w).Encode(body)
}

// DecodeJSON lee el cuerpo exigiendo Content-Type application/json y rechazando
// campos desconocidos, para que un typo en el cliente no se ignore en silencio.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	if !isJSONContentType(r.Header.Get("Content-Type")) {
		return ErrInvalidJSON
	}
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return ErrInvalidJSON
	}
	return nil
}

func isJSONContentType(ct string) bool {
	mediaType, _, err := mime.ParseMediaType(ct)
	if err != nil {
		return false
	}
	return mediaType == "application/json" || mediaType == "text/json"
}
