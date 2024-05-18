package auth

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"

	"github.com/golang-jwt/jwt/v5"
)

const privateKey string = "wFXB414nrRBpPlW0JPuQxZVGDhztONMss4PRAFCKIC9qve9Oh6uksgON11NPBEnw"

func GetPass() (string, error) {
	pass := os.Getenv("TODO_PASSWORD")
	if len(pass) > 0 {
		return pass, nil
	}
	return "", errors.New("password is undefined")
}

func PassHash(pass string) string {
	h := sha256.New()
	h.Write([]byte(pass))
	return fmt.Sprintf("%x", h.Sum(nil))
}

func GetSignedToken() (string, error) {

	secret := []byte(privateKey)
	pass, _ := GetPass()
	passHash := PassHash(pass)

	claims := jwt.MapClaims{
		"hash": passHash,
	}

	jwtToken := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	signedToken, err := jwtToken.SignedString(secret)
	if err != nil {
		return "", err
	}
	return signedToken, nil
}

func VerifyUser(token string) bool {
	jwtToken, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) {
		return []byte(privateKey), nil
	})

	if err != nil {
		return false
	}

	if !jwtToken.Valid {
		return false
	}

	claims, ok := jwtToken.Claims.(jwt.MapClaims)
	if !ok {
		return false
	}

	userPassHashRaw, ok := claims["hash"]
	if !ok {
		return false
	}

	userPassHash, ok := userPassHashRaw.(string)
	if !ok {
		return false
	}
	pass, _ := GetPass()
	passHash := PassHash(pass)

	return passHash == userPassHash
}
