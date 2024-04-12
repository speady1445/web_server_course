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

var (
	accessToken = TokenType{
		Issuer:             "chirpy-access",
		expirationDuration: time.Duration(60*60) * time.Second,
	}
	refreshToken = TokenType{
		Issuer:             "chirpy-refresh",
		expirationDuration: time.Duration(60*60*24*60) * time.Second,
	}
)

type TokenType struct {
	Issuer             string
	expirationDuration time.Duration
}

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

func GetAccessToken(secret string, userID int) (string, error) {
	return getToken(secret, userID, accessToken)
}
func GetRefreshToken(secret string, userID int) (string, error) {
	return getToken(secret, userID, refreshToken)
}

func getToken(secret string, userID int, tokenData TokenType) (string, error) {
	currentUTC := time.Now().UTC()
	expiresAt := currentUTC.Add(tokenData.expirationDuration)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.RegisteredClaims{
		Issuer:    tokenData.Issuer,
		IssuedAt:  jwt.NewNumericDate(currentUTC),
		ExpiresAt: jwt.NewNumericDate(expiresAt),
		Subject:   strconv.Itoa(userID),
	})

	return token.SignedString([]byte(secret))
}

// GetUserID returns the user ID from a signed access token
// Returns error in case token is invalid or missing
func GetUserIDFromAccessToken(secret string, headers http.Header) (userID int, err error) {
	return getUserIDFromToken(secret, headers, accessToken)
}

// GetUserID returns the user ID from a signed refresh token
// Returns error in case token is invalid or missing
func GetUserIDFromRefreshToken(secret string, headers http.Header) (userID int, err error) {
	return getUserIDFromToken(secret, headers, refreshToken)
}

func getUserIDFromToken(secret string, headers http.Header, tokenData TokenType) (userID int, err error) {
	tokenString, err := GetTokenFromHeaders(headers)
	if err != nil {
		return 0, err
	}

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

	if issuer, err := token.Claims.GetIssuer(); err != nil || issuer != tokenData.Issuer {
		return 0, errors.New("invalid token")
	}

	id, err := strconv.Atoi(stringID)
	if err != nil {
		return 0, errors.New("invalid token")
	}

	return id, nil
}

func GetTokenFromHeaders(headers http.Header) (string, error) {
	auth := headers.Get("Authorization")
	if auth == "" {
		return "", errors.New("missing authorization header")
	}

	authSplited := strings.Split(auth, " ")
	if authSplited[0] != "Bearer" || len(authSplited) != 2 {
		return "", errors.New("invalid authorization header")
	}

	return authSplited[1], nil
}
