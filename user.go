package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/speady1445/web_server_course/internals/auth"
	"github.com/speady1445/web_server_course/internals/database"
)

const (
	maxExpiresInSeconds = 24 * 60 * 60
)

type user struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

func (c *apiConfig) handlerAddUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	hash, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dbUser, err := c.db.CreateUser(params.Email, hash)
	if errors.Is(err, database.ErrAlreadyExists) {
		respondWithError(w, http.StatusConflict, err.Error())
		return
	}
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWith(w, http.StatusCreated, user{ID: dbUser.ID, Email: dbUser.Email})
}

func (c *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password         string `json:"password"`
		Email            string `json:"email"`
		ExpiresInSeconds int    `json:"expires_in_seconds"`
	}
	type response struct {
		user

		Token string `json:"token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	dbUser, err := c.db.GetUserByEmail(params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user")
		return
	}

	correctPassword := auth.CheckPassword(params.Password, dbUser.HashedPassword)
	if !correctPassword {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	expiresInSeconds := params.ExpiresInSeconds
	if expiresInSeconds > maxExpiresInSeconds || expiresInSeconds <= 0 {
		expiresInSeconds = maxExpiresInSeconds
	}
	expiresIn := time.Duration(expiresInSeconds) * time.Second

	token, err := auth.GetSignedToken(c.jwtSecret, dbUser.ID, expiresIn)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWith(w, http.StatusOK, response{
		user: user{
			ID:    dbUser.ID,
			Email: dbUser.Email,
		},
		Token: token,
	})
}

func (c *apiConfig) getUserID(r *http.Request) (int, error) {
	id, err := auth.GetUserID(c.jwtSecret, r.Header)
	if err != nil {
		return 0, err
	}
	return id, nil
}

func (c *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := c.getUserID(r)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	err = decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	hashedPassword, err := auth.HashPassword(params.Password)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	dbUser, err := c.db.UpdateUser(id, params.Email, hashedPassword)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWith(w, http.StatusOK, user{ID: dbUser.ID, Email: dbUser.Email})
}
