package seed

import (
	"golang.org/x/crypto/bcrypt"
)

func bcryptHash(s string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(s), 12)
	if err != nil {
		panic("seed: failed to hash demo password: " + err.Error())
	}
	return string(hash)
}