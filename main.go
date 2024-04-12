package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/speady1445/web_server_course/internals/database"
)

const (
	dbPath = "database.json"
)

type apiConfig struct {
	db             *database.DB
	fileserverHits int
	jwtSecret      string
}

func main() {
	const port = "8080"

	godotenv.Load()

	jwtSecret, found := os.LookupEnv("JWT_SECRET")
	if !found {
		fmt.Println("JWT_SECRET not found")
		os.Exit(1)
	}

	err := debug()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	db, err := database.NewDB(dbPath)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	apiCfg := apiConfig{
		db:             db,
		fileserverHits: 0,
		jwtSecret:      jwtSecret,
	}

	mux := http.NewServeMux()
	mux.Handle("/app/*", apiCfg.middlewareMetricsInc(http.StripPrefix("/app/", http.FileServer(http.Dir(".")))))
	mux.HandleFunc("GET /admin/metrics", apiCfg.handlerMetrics)
	mux.HandleFunc("GET /api/reset", apiCfg.handlerReset)
	mux.HandleFunc("GET /api/healthz", healthz)

	mux.HandleFunc("POST /api/users", apiCfg.handlerAddUser)
	mux.HandleFunc("PUT /api/users", apiCfg.handlerUpdateUser)
	mux.HandleFunc("POST /api/login", apiCfg.handlerLogin)
	mux.HandleFunc("POST /api/refresh", apiCfg.handlerRefreshToken)
	mux.HandleFunc("POST /api/revoke", apiCfg.handlerRevokeToken)

	mux.HandleFunc("POST /api/chirps", apiCfg.handlerAddChirp)
	mux.HandleFunc("GET /api/chirps", apiCfg.handlerGetChirps)
	mux.HandleFunc("GET /api/chirps/{chirpid}", apiCfg.handlerGetChirp)
	mux.HandleFunc("DELETE /api/chirps/{chirpid}", apiCfg.handlerDeleteChirp)

	mux.HandleFunc("POST /api/polka/webhooks", apiCfg.handlerPaintUserRed)

	corsMux := middlewareCors(mux)

	server := http.Server{
		Addr:    ":" + port,
		Handler: corsMux,
	}

	fmt.Println("Listening on port " + port)
	server.ListenAndServe()
}

func debug() error {
	dbg := flag.Bool("debug", false, "Enable debug mode")
	flag.Parse()

	if *dbg {
		err := os.Remove(dbPath)
		return err
	}

	return nil
}

func middlewareCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS, PUT, DELETE")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func healthz(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg.fileserverHits++
		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) handlerReset(w http.ResponseWriter, _ *http.Request) {
	cfg.fileserverHits = 0
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Hits reset to 0."))
}

func (cfg *apiConfig) handlerMetrics(w http.ResponseWriter, _ *http.Request) {
	w.Header().Add("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf(`
<html>

<body>
	<h1>Welcome, Chirpy Admin</h1>
	<p>Chirpy has been visited %d times!</p>
</body>

</html>
	`, cfg.fileserverHits)))
}

func respondWithError(w http.ResponseWriter, status int, errMsg string) {
	type errorResponse struct {
		Error string `json:"error"`
	}

	respondWith(w, status, errorResponse{Error: errMsg})
}

func respondWith(w http.ResponseWriter, status int, content interface{}) {
	w.Header().Set("Content-Type", "application/json")
	returnErr, err := json.Marshal(content)
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		w.WriteHeader(500)
	}
	w.WriteHeader(status)
	w.Write(returnErr)
}
