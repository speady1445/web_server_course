package database

import (
	"encoding/json"
	"errors"
	"os"
	"sync"
	"time"
)

var (
	ErrAlreadyExists = errors.New("already exists")
)

type Chirp struct {
	Id       int    `json:"id"`
	AuthorID int    `json:"author_id"`
	Body     string `json:"body"`
}

type User struct {
	ID             int    `json:"id"`
	Email          string `json:"email"`
	HashedPassword string `json:"password"`
}

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps        map[int]Chirp           `json:"chirps"`
	ChirpLastID   int                     `json:"chirp_last_id"`
	Users         map[int]User            `json:"users"`
	RevokedTokens map[string]RevokedToken `json:"revoked_tokens"`
}

type RevokedToken struct {
	Token     string    `json:"token"`
	RevokedAt time.Time `json:"revoked_at"`
}

// NewDB creates a new database connection
// and creates the database file if it doesn't exist
func NewDB(path string) (*DB, error) {
	db := &DB{path: path, mux: &sync.RWMutex{}}
	err := db.ensureDB()
	if err != nil {
		return &DB{}, err
	}
	return db, nil
}

func (db *DB) CreateUser(email string, password string) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	data, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	for _, user := range data.Users {
		if user.Email == email {
			return User{}, ErrAlreadyExists
		}
	}

	id := len(data.Users) + 1
	user := User{
		ID:             id,
		Email:          email,
		HashedPassword: password,
	}

	data.Users[id] = user
	err = db.writeDB(data)
	if err != nil {
		return User{}, err
	}

	return user, nil
}

func (db *DB) GetUserByEmail(email string) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	for _, user := range dbStructure.Users {
		if user.Email == email {
			return user, nil
		}
	}

	return User{}, errors.New("user not found")
}

func (db *DB) UpdateUser(id int, email, hashedPassword string) (User, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return User{}, err
	}

	for dbID, user := range dbStructure.Users {
		if user.ID == id {
			user.Email = email
			user.HashedPassword = hashedPassword

			dbStructure.Users[dbID] = user

			return user, db.writeDB(dbStructure)
		}
	}

	return User{}, errors.New("user not found")
}

// CreateChirp creates a new chirp and saves it to disk
func (db *DB) CreateChirp(authorID int, body string) (Chirp, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	dbStructure.ChirpLastID++
	id := dbStructure.ChirpLastID
	chirp := Chirp{
		Id:       id,
		AuthorID: authorID,
		Body:     body,
	}

	dbStructure.Chirps[id] = chirp
	err = db.writeDB(dbStructure)
	if err != nil {
		return Chirp{}, err
	}

	return chirp, nil
}

func (db *DB) DeleteChirp(id int) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return err
	}

	delete(dbStructure.Chirps, id)

	return db.writeDB(dbStructure)
}

// GetChirps returns all chirps in the database
func (db *DB) GetChirps() ([]Chirp, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return []Chirp{}, err
	}

	chirps := make([]Chirp, 0, len(dbStructure.Chirps))
	for _, chirp := range dbStructure.Chirps {
		chirps = append(chirps, chirp)
	}

	return chirps, nil
}

// GetChirp returns a single chirp by ID
func (db *DB) GetChirp(id int) (Chirp, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return Chirp{}, err
	}

	chirp, exists := dbStructure.Chirps[id]
	if !exists {
		return Chirp{}, errors.New("chirp does not exist")
	}

	return chirp, nil
}

// ensureDB creates a new database file if it doesn't exist
func (db *DB) ensureDB() error {
	if _, err := os.Stat(db.path); !errors.Is(err, os.ErrNotExist) {
		return nil
	}

	emptyDB := DBStructure{
		Chirps:        map[int]Chirp{},
		ChirpLastID:   0,
		Users:         map[int]User{},
		RevokedTokens: map[string]RevokedToken{},
	}
	db.writeDB(emptyDB)
	return nil
}

func (db *DB) AddRevokedToken(tokenString string) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return err
	}

	token := RevokedToken{
		Token:     tokenString,
		RevokedAt: time.Now().UTC(),
	}
	dbStructure.RevokedTokens[tokenString] = token

	return db.writeDB(dbStructure)
}

func (db *DB) IsTokenRevoked(tokenString string) (bool, error) {
	db.mux.Lock()
	defer db.mux.Unlock()

	dbStructure, err := db.loadDB()
	if err != nil {
		return false, err
	}

	_, exists := dbStructure.RevokedTokens[tokenString]
	if exists {
		return true, nil
	}

	return false, nil
}

// loadDB reads the database file into memory
func (db *DB) loadDB() (DBStructure, error) {
	data := DBStructure{}

	file, err := os.ReadFile(db.path)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(file, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

// writeDB writes the database file to disk
func (db *DB) writeDB(dbStructure DBStructure) error {
	jsonData, err := json.Marshal(dbStructure)
	if err != nil {
		return err
	}

	return os.WriteFile(db.path, jsonData, 0600)
}
