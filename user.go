package main

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/speady1445/web_server_course/internals/auth"
	"github.com/speady1445/web_server_course/internals/database"
)

type responseUser struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	IsChirpyRed bool   `json:"is_chirpy_red"`
}

func dbUserToResponseUser(dbUser database.User) responseUser {
	return responseUser{
		ID:          dbUser.ID,
		Email:       dbUser.Email,
		IsChirpyRed: dbUser.IsChirpyRed,
	}
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

	respondWith(w, http.StatusCreated, dbUserToResponseUser(dbUser))
}

func (c *apiConfig) handlerLogin(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Password string `json:"password"`
		Email    string `json:"email"`
	}
	type response struct {
		responseUser
		AccessToken  string `json:"token"`
		RefreshToken string `json:"refresh_token"`
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

	accessToken, err := auth.GetAccessToken(c.jwtSecret, dbUser.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	refreshToken, err := auth.GetRefreshToken(c.jwtSecret, dbUser.ID)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWith(w, http.StatusOK, response{
		responseUser: dbUserToResponseUser(dbUser),
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	})
}

func (c *apiConfig) handlerUpdateUser(w http.ResponseWriter, r *http.Request) {
	id, err := auth.GetUserIDFromAccessToken(c.jwtSecret, r.Header)
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

	respondWith(w, http.StatusOK, dbUserToResponseUser(dbUser))
}

func (c *apiConfig) handlerRefreshToken(w http.ResponseWriter, r *http.Request) {
	type response struct {
		Token string `json:"token"`
	}

	userId, err := auth.GetUserIDFromRefreshToken(c.jwtSecret, r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	refreshToken, err := auth.GetTokenFromHeaders(r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	isRevoked, err := c.db.IsTokenRevoked(refreshToken)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if isRevoked {
		respondWithError(w, http.StatusUnauthorized, "Token already revoked")
		return
	}

	newAccessToken, err := auth.GetAccessToken(c.jwtSecret, userId)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWith(w, http.StatusOK, response{Token: newAccessToken})
}

func (c *apiConfig) handlerRevokeToken(w http.ResponseWriter, r *http.Request) {
	_, err := auth.GetUserIDFromRefreshToken(c.jwtSecret, r.Header)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, err.Error())
		return
	}

	token, err := auth.GetTokenFromHeaders(r.Header)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Could not find token")
		return
	}

	err = c.db.AddRevokedToken(token)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Could not revoke token")
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (c *apiConfig) handlerPaintUserRed(w http.ResponseWriter, r *http.Request) {
	type parameters struct {
		Event string `json:"event"`
		Data  struct {
			UserID int `json:"user_id"`
		} `json:"data"`
	}

	decoder := json.NewDecoder(r.Body)
	params := parameters{}
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid input")
		return
	}

	if params.Event != "user.upgraded" {
		w.WriteHeader(http.StatusOK)
		return
	}

	err = c.db.PaintUserRed(params.Data.UserID)
	if err != nil {
		if errors.Is(err, database.ErrNotExists) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "Could not update user")
		return
	}

	w.WriteHeader(http.StatusOK)
}
