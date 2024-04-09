package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/speady1445/web_server_course/internals/auth"
	"github.com/speady1445/web_server_course/internals/database"
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

	dbUser, err := c.db.GetUser(params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not find user")
		return
	}

	correctPassword := auth.CheckPassword(params.Password, dbUser.HashedPassword)
	if !correctPassword {
		respondWithError(w, http.StatusUnauthorized, "Invalid credentials")
		return
	}

	respondWith(w, http.StatusOK, user{ID: dbUser.ID, Email: dbUser.Email})
}
