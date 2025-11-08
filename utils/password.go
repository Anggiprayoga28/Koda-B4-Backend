package utils

import (
	"github.com/matthewhartstonge/argon2"
)

func HashPassword(password string) (string, error) {
	argon := argon2.DefaultConfig()
	encoded, err := argon.HashEncoded([]byte(password))
	if err != nil {
		return "", err
	}
	return string(encoded), nil
}

func VerifyPassword(encodedHash, password string) (bool, error) {
	ok, err := argon2.VerifyEncoded([]byte(password), []byte(encodedHash))
	if err != nil {
		return false, err
	}
	return ok, nil
}
