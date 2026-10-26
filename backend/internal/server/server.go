package server

import (
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	AllowedOrigins   []string
	UploadDir        string
	WebDir           string
	PaymentsMode     string
	TermsVersion     string
	RegistrationFee  int64
	RegistrationDays int
	RevealPrice      int64
	DeliveryFee      int64
	BoostDays        int
	DisableRateLimit bool
	AI               AIConfig
	Storage          SupabaseStorage
}

type Server struct {
	db  *pgxpool.Pool
	cfg Config
	ai  *AIClient
	mux *http.ServeMux
	rl  *rateLimiter
}

func New(db *pgxpool.Pool, cfg Config) *Server {
	if cfg.PaymentsMode == "free" {
		cfg.RegistrationFee, cfg.RevealPrice = 0, 0
	}
	s := &Server{db: db, cfg: cfg, ai: NewAIClient(cfg.AI), mux: http.NewServeMux(), rl: newRateLimiter()}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler { return securityHeaders(s.cors(s.mux)) }

func (s *Server) routes() {
	m := s.mux
	m.HandleFunc("GET /api/health", s.health)
	m.HandleFunc("GET /api/config", s.publicConfig)
	m.HandleFunc("GET /api/categories", pub(s.listCategories))

	m.HandleFunc("POST /api/auth/register", pub(s.register))
	m.HandleFunc("POST /api/auth/login", pub(s.login))
	m.HandleFunc("POST /api/auth/logout", s.authed(s.logout))
	m.HandleFunc("GET /api/me", s.authed(s.me))
	m.HandleFunc("DELETE /api/me", s.authed(s.deleteAccount))

	m.HandleFunc("GET /api/me/technician-profile", s.authed(s.getTechnicianProfile, "technician"))
	m.HandleFunc("PUT /api/me/technician-profile", s.authed(s.putTechnicianProfile, "technician"))
	m.HandleFunc("GET /api/technicians", pub(s.searchTechnicians))
	m.HandleFunc("GET /api/technicians/{id}", pub(s.getTechnician))

	m.HandleFunc("POST /api/estimate", s.authed(s.previewEstimate))
	m.HandleFunc("GET /api/estimate/typical", pub(s.typicalEstimate))
	m.HandleFunc("GET /api/stats/categories", pub(s.categoryStats))
	m.HandleFunc("GET /api/posts/recent", pub(s.recentPosts))
	m.HandleFunc("POST /api/posts", s.authed(s.createPost, "customer"))
	m.HandleFunc("GET /api/posts/mine", s.authed(s.myPosts, "customer"))
	m.HandleFunc("GET /api/posts/nearby", s.authed(s.nearbyPosts, "technician"))
	m.HandleFunc("GET /api/posts/{id}", s.authed(s.getPost))
	m.HandleFunc("POST /api/posts/{id}/reveal", s.authed(s.revealContact, "technician"))
	m.HandleFunc("POST /api/posts/{id}/close", s.authed(s.closePost, "customer"))
	m.HandleFunc("GET /api/posts/{id}/viewers", s.authed(s.postViewers, "customer"))

	m.HandleFunc("GET /api/stalls", pub(s.listStalls))
	m.HandleFunc("GET /api/me/seller-profile", s.authed(s.getSellerProfile, "seller"))
	m.HandleFunc("PUT /api/me/seller-profile", s.authed(s.putSellerProfile, "seller"))
	m.HandleFunc("GET /api/listings", pub(s.searchListings))
	m.HandleFunc("GET /api/listings/mine", s.authed(s.myListings, "seller"))
	m.HandleFunc("GET /api/listings/{id}", pub(s.getListing))
	m.HandleFunc("POST /api/listings", s.authed(s.createListing, "seller"))
	m.HandleFunc("PATCH /api/listings/{id}", s.authed(s.updateListing, "seller"))

	m.HandleFunc("POST /api/orders", s.authed(s.createOrder, "customer", "technician"))
	m.HandleFunc("GET /api/orders/mine", s.authed(s.myOrders))
	m.HandleFunc("GET /api/orders/incoming", s.authed(s.incomingOrders, "seller"))
	m.HandleFunc("PATCH /api/orders/{id}", s.authed(s.updateOrder))

	m.HandleFunc("GET /api/wallet", s.authed(s.wallet))
	m.HandleFunc("POST /api/wallet/topup", s.authed(s.topUp))
	m.HandleFunc("POST /api/registration/pay", s.authed(s.payRegistration, "technician", "seller"))
	m.HandleFunc("POST /api/boosts", s.authed(s.buyBoost, "technician", "seller"))
	m.HandleFunc("GET /api/leaderboard", pub(s.leaderboard))

	m.HandleFunc("POST /api/reports", s.authed(s.createReport))
	m.HandleFunc("GET /api/admin/reports", s.authed(s.adminReports, "admin"))
	m.HandleFunc("POST /api/admin/reports/{id}/resolve", s.authed(s.adminResolveReport, "admin"))
	m.HandleFunc("GET /api/admin/reveals", s.authed(s.adminReveals, "admin"))
	m.HandleFunc("GET /api/admin/flagged-posts", s.authed(s.adminFlaggedPosts, "admin"))
	m.HandleFunc("POST /api/admin/stalls", s.authed(s.adminCreateStall, "admin"))
	m.HandleFunc("PATCH /api/admin/stalls/{id}", s.authed(s.adminUpdateStall, "admin"))

	m.Handle("GET /uploads/", http.StripPrefix("/uploads/", noListing(http.FileServer(http.Dir(s.cfg.UploadDir)))))
	if s.cfg.WebDir != "" {
		m.Handle("GET /", s.webHandler())
	}
}

func pub(h func(http.ResponseWriter, *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := h(w, r); err != nil {
			writeError(w, err)
		}
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	if err := s.db.Ping(r.Context()); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "error", "database": "unreachable"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "database": "connected"})
}

func (s *Server) publicConfig(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"paymentsMode":     s.cfg.PaymentsMode,
		"termsVersion":     s.cfg.TermsVersion,
		"registrationFee":  s.cfg.RegistrationFee,
		"registrationDays": s.cfg.RegistrationDays,
		"revealPrice":      s.cfg.RevealPrice,
		"deliveryFee":      s.cfg.DeliveryFee,
		"boostDays":        s.cfg.BoostDays,
		"aiEnabled":        s.ai.Enabled(),
	})
}

func (s *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && (slices.Contains(s.cfg.AllowedOrigins, origin) || slices.Contains(s.cfg.AllowedOrigins, "*")) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func noListing(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type httpError struct {
	status int
	msg    string
}

func (e *httpError) Error() string { return e.msg }

func errStatus(status int, format string, args ...any) error {
	return &httpError{status: status, msg: fmt.Sprintf(format, args...)}
}

func badRequest(format string, args ...any) error {
	return errStatus(http.StatusBadRequest, format, args...)
}

func writeError(w http.ResponseWriter, err error) {
	var he *httpError
	if errors.As(err, &he) {
		writeJSON(w, he.status, map[string]string{"error": he.msg})
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Not found"})
		return
	}
	log.Printf("internal error: %v", err)
	writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Something went wrong"})
}

func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("could not encode JSON response: %v", err)
	}
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return badRequest("Invalid JSON request: %v", err)
	}
	return nil
}

func pathID(r *http.Request) (int64, error) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		return 0, badRequest("Invalid id")
	}
	return id, nil
}

func queryFloat(r *http.Request, key string) (float64, bool, error) {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0, false, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, false, badRequest("Invalid %s", key)
	}
	return f, true, nil
}

// queryPoint reads optional lat/lng query params; both or neither must be set.
func queryPoint(r *http.Request) (*Point, error) {
	lat, okLat, err := queryFloat(r, "lat")
	if err != nil {
		return nil, err
	}
	lng, okLng, err := queryFloat(r, "lng")
	if err != nil {
		return nil, err
	}
	if !okLat && !okLng {
		return nil, nil
	}
	p := &Point{Lat: lat, Lng: lng}
	return p, p.validate()
}

type Point struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

func (p *Point) validate() error {
	if p == nil {
		return badRequest("Location is required")
	}
	if p.Lat < -90 || p.Lat > 90 || p.Lng < -180 || p.Lng > 180 || (p.Lat == 0 && p.Lng == 0) {
		return badRequest("Invalid location")
	}
	return nil
}

func trimmed(s string, field string, min, max int) (string, error) {
	s = strings.TrimSpace(s)
	n := len([]rune(s))
	if n < min {
		if min == 1 {
			return "", badRequest("%s is required", field)
		}
		return "", badRequest("%s must be at least %d characters", field, min)
	}
	if n > max {
		return "", badRequest("%s must not exceed %d characters", field, max)
	}
	return s, nil
}

//go:embed migrations/*.sql
var migrationFS embed.FS

func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	if _, err := db.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	names, err := fs.Glob(migrationFS, "migrations/*.sql")
	if err != nil {
		return err
	}
	sort.Strings(names)
	for _, name := range names {
		base := filepath.Base(name)
		var exists bool
		if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`, base).Scan(&exists); err != nil {
			return err
		}
		if exists {
			continue
		}
		sqlBytes, err := migrationFS.ReadFile(name)
		if err != nil {
			return err
		}
		err = pgx.BeginFunc(ctx, db, func(tx pgx.Tx) error {
			if _, err := tx.Exec(ctx, string(sqlBytes)); err != nil {
				return err
			}
			_, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, base)
			return err
		})
		if err != nil {
			return fmt.Errorf("migration %s: %w", base, err)
		}
		log.Printf("applied migration %s", base)
	}
	return nil
}

func ensureDir(dir string) error { return os.MkdirAll(dir, 0o755) }

const contentSecurityPolicy = "default-src 'self'; script-src 'self' https://unpkg.com; style-src 'self' 'unsafe-inline' https://unpkg.com; " +
	"img-src 'self' data: blob: https:; media-src 'self' blob: https:; connect-src 'self' https:; font-src 'self' data:; " +
	"object-src 'none'; base-uri 'self'; form-action 'self'; frame-ancestors 'none'"

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "geolocation=(self), camera=(), microphone=()")
		h.Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}

// webHandler serves the static site and answers unknown paths with 404.html.
func (s *Server) webHandler() http.Handler {
	files := http.FileServer(http.Dir(s.cfg.WebDir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := path.Clean("/" + r.URL.Path)
		if p != "/" {
			if _, err := os.Stat(filepath.Join(s.cfg.WebDir, filepath.FromSlash(p))); err != nil {
				body, err := os.ReadFile(filepath.Join(s.cfg.WebDir, "404.html"))
				if err != nil {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write(body)
				return
			}
		}
		files.ServeHTTP(w, r)
	})
}
