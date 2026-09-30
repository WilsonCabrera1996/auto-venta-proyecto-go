// Package crypto contiene los adaptadores de criptografia. El dominio declara
// el puerto Hasher; aca esta la implementacion real.
package crypto

import (
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// BcryptHasher cumple domain.Hasher. bcrypt es la unica pieza criptografica que
// se importa de fuera: la stdlib no trae hashing de passwords y hacerlo a mano
// es exactamente el error que hay que evitar.
type BcryptHasher struct {
	cost int
}

func NewBcryptHasher() *BcryptHasher {
	return &BcryptHasher{cost: bcrypt.DefaultCost}
}

func (h *BcryptHasher) Hash(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("bcrypt hash: %w", err)
	}
	return string(b), nil
}

func (h *BcryptHasher) Compare(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return fmt.Errorf("bcrypt compare: %w", err)
	}
	return nil
}
