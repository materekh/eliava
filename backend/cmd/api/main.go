package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Technician struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Specialty       string    `json:"specialty"`
	City            string    `json:"city"`
	ServiceRadiusKM int       `json:"serviceRadiusKm"`
	Description     string    `json:"description"`
	CreatedAt       time.Time `json:"createdAt"`
}

func main() {
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		log.Fatal("DB_PASSWORD is not set. Set it in PowerShell before starting the server.")
	}

	config, err := pgxpool.ParseConfig(
		"host=localhost port=5432 user=postgres dbname=eliava sslmode=disable",
	)
	if err != nil {
		log.Fatalf("Could not parse database config: %v", err)
	}

	config.ConnConfig.Password = password

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		cancel()
		log.Fatalf("Could not create database pool: %v", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		cancel()
		log.Fatalf("Could not connect to PostgreSQL: %v", err)
	}

	cancel()
	defer pool.Close()

	log.Println("Connected to PostgreSQL database: eliava")

	mux := http.NewServeMux()

	mux.Handle("/api/health", cors(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, `{"error":"Method not allowed"}`, http.StatusMethodNotAllowed)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{
			"status":   "ok",
			"database": "connected",
		})
	})))

	mux.Handle("/api/technicians", cors(techniciansHandler(pool)))

	log.Println("RepairLink API running at http://localhost:8080")
	log.Fatal(http.ListenAndServe("localhost:8080", mux))
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		if origin == "http://localhost:3000" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func techniciansHandler(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listTechnicians(w, r, pool)
		case http.MethodPost:
			createTechnician(w, r, pool)
		default:
			w.Header().Set("Allow", "GET, POST, OPTIONS")
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
				"error": "Method not allowed",
			})
		}
	})
}

func listTechnicians(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) {
	rows, err := pool.Query(r.Context(), `
		SELECT id, name, specialty, city, service_radius_km, description, created_at
		FROM technicians
		ORDER BY id DESC
	`)
	if err != nil {
		log.Printf("Could not query technicians: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Could not load technicians",
		})
		return
	}
	defer rows.Close()

	technicians := make([]Technician, 0)

	for rows.Next() {
		var t Technician
		if err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.Specialty,
			&t.City,
			&t.ServiceRadiusKM,
			&t.Description,
			&t.CreatedAt,
		); err != nil {
			log.Printf("Could not read technician: %v", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{
				"error": "Could not read technicians",
			})
			return
		}
		technicians = append(technicians, t)
	}

	if err := rows.Err(); err != nil {
		log.Printf("Error iterating technicians: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Could not load technicians",
		})
		return
	}

	writeJSON(w, http.StatusOK, technicians)
}

func createTechnician(w http.ResponseWriter, r *http.Request, pool *pgxpool.Pool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	var input struct {
		Name            string `json:"name"`
		Specialty       string `json:"specialty"`
		City            string `json:"city"`
		ServiceRadiusKM int    `json:"serviceRadiusKm"`
		Description     string `json:"description"`
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Invalid JSON request",
		})
		return
	}

	input.Name = strings.TrimSpace(input.Name)
	input.Specialty = strings.TrimSpace(input.Specialty)
	input.City = strings.TrimSpace(input.City)
	input.Description = strings.TrimSpace(input.Description)

	if input.Name == "" || input.Specialty == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Name and specialty are required",
		})
		return
	}

	if len(input.Name) > 100 || len(input.Specialty) > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Name and specialty must not exceed 100 characters",
		})
		return
	}

	if input.City == "" {
		input.City = "Tbilisi"
	}
	if len(input.City) > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "City must not exceed 100 characters",
		})
		return
	}

	if input.ServiceRadiusKM == 0 {
		input.ServiceRadiusKM = 10
	}
	if input.ServiceRadiusKM < 1 || input.ServiceRadiusKM > 500 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "Service radius must be between 1 and 500 km",
		})
		return
	}

	var t Technician

	err := pool.QueryRow(r.Context(), `
		INSERT INTO technicians
			(name, specialty, city, service_radius_km, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, specialty, city, service_radius_km, description, created_at
	`,
		input.Name,
		input.Specialty,
		input.City,
		input.ServiceRadiusKM,
		input.Description,
	).Scan(
		&t.ID,
		&t.Name,
		&t.Specialty,
		&t.City,
		&t.ServiceRadiusKM,
		&t.Description,
		&t.CreatedAt,
	)

	if err != nil {
		log.Printf("Could not create technician: %v", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "Could not save technician",
		})
		return
	}

	writeJSON(w, http.StatusCreated, t)
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Could not encode JSON response: %v", err)
	}
}
