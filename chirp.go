package main

import (
	"encoding/json"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/speady1445/web_server_course/internals/auth"
	"github.com/speady1445/web_server_course/internals/database"
)

type responseChirp struct {
	ID       int    `json:"id"`
	AuthorID int    `json:"author_id"`
	Body     string `json:"body"`
}

func dbChirpToResponseChirp(dbChirp database.Chirp) responseChirp {
	return responseChirp{
		ID:       dbChirp.Id,
		AuthorID: dbChirp.AuthorID,
		Body:     dbChirp.Body,
	}
}

func (c *apiConfig) handlerAddChirp(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Body  string `json:"body"`
		Token string `json:"token"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	userID, err := auth.GetUserIDFromAccessToken(c.jwtSecret, r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	if len(params.Body) > 140 {
		respondWithError(w, http.StatusBadRequest, "Chirp is too long")
		return
	}

	dbChirp, err := c.db.CreateChirp(userID, cleanedChirpMessage(params.Body))
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWith(w, http.StatusCreated, dbChirpToResponseChirp(dbChirp))
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

	chirps := make([]responseChirp, 0, len(dbChirps))
	for _, dbChirp := range dbChirps {
		chirps = append(chirps, dbChirpToResponseChirp(dbChirp))
	}

	slices.SortFunc(chirps, func(a, b responseChirp) int {
		return a.ID - b.ID
	})

	respondWith(w, http.StatusOK, chirps)
}

func (c *apiConfig) handlerGetChirp(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("chirpid")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp id.")
		return
	}

	dbChirp, err := c.db.GetChirp(id)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found")
		return
	}

	respondWith(w, http.StatusOK, dbChirpToResponseChirp(dbChirp))
}

func (c *apiConfig) handlerDeleteChirp(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("chirpid")
	inputID, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid chirp id.")
		return
	}

	authorID, err := auth.GetUserIDFromAccessToken(c.jwtSecret, r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid access token.")
		return
	}

	chirp, err := c.db.GetChirp(inputID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "Chirp not found.")
		return
	}

	if authorID != chirp.AuthorID {
		respondWithError(w, http.StatusForbidden, "You can only delete your own chirps.")
		return
	}

	err = c.db.DeleteChirp(inputID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Error deleting chirp.")
		return
	}

	w.WriteHeader(http.StatusOK)
}
