// Package middleware contiene los middlewares transversales del servicio.
package middleware

import "net/http"

type Middleware func(http.Handler) http.Handler

// Chain aplica los middlewares de izquierda a derecha: el primero es el mas
// externo (se ejecuta primero) y el ultimo es el que envuelve al handler.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}
