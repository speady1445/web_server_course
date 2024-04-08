package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
)

type chirp struct {
	ID   int    `json:"id"`
	Body string `json:"body"`
}

func (c *apiConfig) handlerAddChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body string `json:"body"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)

	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	dbChirp, err := c.db.CreateChirp(cleanedChirpMessage(params.Body))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWith(w, http.StatusCreated, chirp{ID: dbChirp.Id, Body: dbChirp.Body})
}

func cleanedChirpMessage(text string) string {
	bannedWords := map[string]struct{}{
		"kerfuffle": {},
		"sharbert":  {},
		"fornax":    {},
	}

	words := strings.Split(text, " ")
	for i, word := range words {
		if _, ok := bannedWords[strings.ToLower(word)]; ok {
			words[i] = "****"
		}
	}
	return strings.Join(words, " ")
}

func (c *apiConfig) handlerGetChirps(w http.ResponseWriter, _ *http.Request) {
	dbChirps, err := c.db.GetChirps()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	chirps := make([]chirp, 0, len(dbChirps))
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, chirp{ID: dbChirp.Id, Body: dbChirp.Body})
	}

	slices.SortFunc(chirps, func(a, b chirp) int {
		return a.ID - b.ID
	})

	respondWith(w, http.StatusOK, chirps)
}
