package auth

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func GetSignedToken(secret string, userID int, expiresIn time.Duration) (string, error) {
	currentUTC := time.Now().UTC()
	expiresAt := currentUTC.Add(expiresIn)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    "chirpy",
		IssuedAt:  jwt.NewNumericDate(currentUTC),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Subject:   strconv.Itoa(userID),
	})

	return token.SignedString([]byte(secret))
}

// GetUserID returns the user ID from a signed token
// If err is not nil, the token is invalid or missing
func GetUserID(secret string, headers http.Header) (userID int, err error) {
	tokenString := strings.TrimPrefix(headers.Get("Authorization"), "Bearer ")
	claim := &jwt.RegisteredClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claim, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return 0, err
	}

	stringID, err := token.Claims.GetSubject()
	if err != nil {
		return 0, errors.New("invalid token")
	}

	id, err := strconv.Atoi(stringID)
	if err != nil {
		return 0, errors.New("invalid token")
	}

	return id, nil
}
