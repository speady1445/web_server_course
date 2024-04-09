package main

import (
	"encoding/json"
	"net/http"
)

type user struct {
	ID    int    `json:"id"`
	Email string `json:"email"`
}

func (c *apiConfig) handlerAddUser(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}

	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	dbUser, err := c.db.CreateUser(params.Email)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWith(w, http.StatusCreated, user{ID: dbUser.Id, Email: dbUser.Email})
}
