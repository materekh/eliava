package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/eliava/backend/internal/server"
	"github.com/jackc/pgx/v5/pgxpool"
)

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil || n < 0 {
		log.Fatalf("%s must be a non-negative whole number", key)
	}
	return n
}

// databaseConfig uses DATABASE_URL when set (hosted databases), otherwise the local DB_* variables.
func databaseConfig() *pgxpool.Config {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		cfg, err := pgxpool.ParseConfig(url)
		if err != nil {
			log.Fatalf("Could not parse DATABASE_URL: %v", err)
		}
		return cfg
	}
	password := os.Getenv("DB_PASSWORD")
	if password == "" {
		log.Fatal("Set DATABASE_URL, or DB_PASSWORD for a local PostgreSQL database.")
	}
	cfg, err := pgxpool.ParseConfig("host=" + env("DB_HOST", "localhost") + " port=" + env("DB_PORT", "5432") +
		" user=" + env("DB_USER", "postgres") + " dbname=" + env("DB_NAME", "eliava") + " sslmode=" + env("DB_SSLMODE", "disable"))
	if err != nil {
		log.Fatalf("Could not parse database config: %v", err)
	}
	cfg.ConnConfig.Password = password
	return cfg
}

func defaultWebDir() string {
	for _, dir := range []string{"web", "../web", "../../web"} {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	pool, err := pgxpool.NewWithConfig(ctx, databaseConfig())
	if err != nil {
		log.Fatalf("Could not create database pool: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("Could not connect to PostgreSQL: %v", err)
	}
	if err := server.Migrate(ctx, pool); err != nil {
		log.Fatalf("Could not migrate database: %v", err)
	}
	adminEmail := strings.ToLower(env("ADMIN_EMAIL", ""))
	if err := server.BootstrapAdmin(ctx, pool, adminEmail, os.Getenv("ADMIN_PASSWORD")); err != nil {
		log.Fatalf("Could not set up admin: %v", err)
	}
	if adminEmail != "" && os.Getenv("ADMIN_PASSWORD") == "" {
		log.Printf("ADMIN_EMAIL is set without ADMIN_PASSWORD: no admin account was created or promoted")
	}
	cancel()
	defer pool.Close()

	cfg := server.Config{
		AllowedOrigins:   strings.Split(env("ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8080"), ","),
		UploadDir:        env("UPLOAD_DIR", "uploads"),
		WebDir:           env("WEB_DIR", defaultWebDir()),
		PaymentsMode:     env("PAYMENTS_MODE", "free"),
		TermsVersion:     env("TERMS_VERSION", "2026-10-draft"),
		RegistrationFee:  envInt("REGISTRATION_FEE_TETRI", 2000),
		RegistrationDays: int(envInt("REGISTRATION_DAYS", 30)),
		RevealPrice:      envInt("REVEAL_PRICE_TETRI", 200),
		DeliveryFee:      envInt("DELIVERY_FEE_TETRI", 500),
		BoostDays:        int(envInt("BOOST_DAYS", 30)),
		AI: server.AIConfig{
			APIKey:  os.Getenv("AI_API_KEY"),
			BaseURL: os.Getenv("AI_BASE_URL"),
			Model:   os.Getenv("AI_MODEL"),
		},
		Storage: server.SupabaseStorage{
			URL:        os.Getenv("SUPABASE_URL"),
			ServiceKey: os.Getenv("SUPABASE_SERVICE_KEY"),
			Bucket:     env("SUPABASE_BUCKET", "media"),
		},
	}
	for i := range cfg.AllowedOrigins {
		cfg.AllowedOrigins[i] = strings.TrimSpace(cfg.AllowedOrigins[i])
	}
	if cfg.Storage.Enabled() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := cfg.Storage.EnsureBucket(ctx); err != nil {
			log.Fatalf("Supabase Storage: %v", err)
		}
		cancel()
	} else if err := os.MkdirAll(cfg.UploadDir, 0o755); err != nil {
		log.Fatalf("Could not create upload dir: %v", err)
	}

	addr := env("ADDR", ":"+env("PORT", "8080"))
	srv := &http.Server{
		Addr:              addr,
		Handler:           server.New(pool, cfg).Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	log.Printf("Eliava API running on %s (payments: %s, AI: %t, Supabase storage: %t, web: %q)", addr, cfg.PaymentsMode, cfg.AI.APIKey != "", cfg.Storage.Enabled(), cfg.WebDir)
	log.Fatal(srv.ListenAndServe())
}
