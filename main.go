package main

import (
	"awesomeProject/internal/database"
	"database/sql"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {

	godotenv.Load()
	dbURL := os.Getenv("DB_URL")

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Printf("Error opening database: %s", err)
	}
	defer db.Close()

	router := http.NewServeMux()
	apiCfg := apiConfig{
		dbQueries: database.New(db),
	}

	fileserver := http.FileServer(http.Dir("."))
	appHandler := http.StripPrefix("/app/", fileserver)

	router.Handle("/app/", apiCfg.middlewareMetricsInc(appHandler))

	router.HandleFunc("GET /api/healthz", apiCfg.checkServer)
	router.HandleFunc("GET /admin/metrics", apiCfg.getHits)
	router.HandleFunc("POST /admin/reset", apiCfg.reset)
	router.HandleFunc("POST /api/validate", apiCfg.handleJSON)
	router.HandleFunc("POST /api/users", apiCfg.createUser)

	server := http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	waitG := sync.WaitGroup{}
	waitG.Add(1)
	go server.ListenAndServe()
	log.Printf("Server started and listening on %s", server.Addr)

	waitG.Wait()
}

type User struct {
	ID        uuid.UUID `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Email     string    `json:"email"`
}
