package server

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/mail"
	"slices"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

const sessionTTL = 30 * 24 * time.Hour

type User struct {
	ID                    int64      `json:"id"`
	Role                  string     `json:"role"`
	Name                  string     `json:"name"`
	Email                 string     `json:"email"`
	Phone                 string     `json:"phone"`
	BalanceTetri          int64      `json:"balanceTetri"`
	RegistrationPaidUntil *time.Time `json:"registrationPaidUntil"`
	RegistrationActive    bool       `json:"registrationActive"`
	CreatedAt             time.Time  `json:"createdAt"`
}

const userColumns = `id, role, name, email, phone, balance_tetri, registration_paid_until, created_at`

func scanUser(row pgx.Row) (*User, error) {
	var u User
	if err := row.Scan(&u.ID, &u.Role, &u.Name, &u.Email, &u.Phone, &u.BalanceTetri, &u.RegistrationPaidUntil, &u.CreatedAt); err != nil {
		return nil, err
	}
	u.RegistrationActive = u.Role == "admin" || (u.RegistrationPaidUntil != nil && u.RegistrationPaidUntil.After(time.Now()))
	return &u, nil
}

type authedHandler func(http.ResponseWriter, *http.Request, *User) error

// authed requires a valid session; if roles are given the user must have one of them (admins pass every check).
func (s *Server) authed(h authedHandler, roles ...string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u, err := s.currentUser(r)
		if err != nil {
			writeError(w, err)
			return
		}
		if len(roles) > 0 && u.Role != "admin" && !slices.Contains(roles, u.Role) {
			writeError(w, errStatus(http.StatusForbidden, "This action is only for %s accounts", strings.Join(roles, " / ")))
			return
		}
		if err := h(w, r, u); err != nil {
			writeError(w, err)
		}
	}
}

func (s *Server) currentUser(r *http.Request) (*User, error) {
	token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok || token == "" {
		return nil, errStatus(http.StatusUnauthorized, "Please log in")
	}
	u, err := scanUser(s.db.QueryRow(r.Context(), `
		SELECT `+prefixed("u.", userColumns)+`
		FROM sessions s JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1 AND s.expires_at > now()`, hashToken(token)))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errStatus(http.StatusUnauthorized, "Your session expired, please log in again")
	}
	return u, err
}

func prefixed(prefix, columns string) string {
	parts := strings.Split(columns, ", ")
	for i, p := range parts {
		parts[i] = prefix + p
	}
	return strings.Join(parts, ", ")
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *Server) newSession(r *http.Request, userID int64) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	token := hex.EncodeToString(b)
	_, err := s.db.Exec(r.Context(), `INSERT INTO sessions (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		hashToken(token), userID, time.Now().Add(sessionTTL))
	return token, err
}

func normalizePhone(phone string) (string, error) {
	phone = strings.TrimSpace(phone)
	digits := 0
	for i, c := range phone {
		switch {
		case unicode.IsDigit(c):
			digits++
		case c == '+' && i == 0, c == ' ', c == '-':
		default:
			return "", badRequest("Phone number may only contain digits, spaces, dashes and a leading +")
		}
	}
	if digits < 9 || digits > 15 {
		return "", badRequest("Enter a valid phone number")
	}
	return phone, nil
}

func (s *Server) register(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Role        string `json:"role"`
		Name        string `json:"name"`
		Email       string `json:"email"`
		Phone       string `json:"phone"`
		Password    string `json:"password"`
		AcceptTerms bool   `json:"acceptTerms"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if !slices.Contains([]string{"customer", "technician", "seller"}, in.Role) {
		return badRequest("Choose an account type: customer, technician or seller")
	}
	if !in.AcceptTerms {
		return badRequest("You must accept the Terms & Conditions")
	}
	name, err := trimmed(in.Name, "Name", 2, 100)
	if err != nil {
		return err
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return badRequest("Enter a valid email address")
	}
	phone, err := normalizePhone(in.Phone)
	if err != nil {
		return err
	}
	if len(in.Password) < 8 || len(in.Password) > 72 {
		return badRequest("Password must be 8-72 characters")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	role := in.Role
	if s.cfg.AdminEmail != "" && strings.EqualFold(email, s.cfg.AdminEmail) {
		role = "admin"
	}

	var u *User
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var err error
		u, err = scanUser(tx.QueryRow(r.Context(), `
			INSERT INTO users (role, name, email, phone, password_hash, terms_version)
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING `+userColumns, role, name, email, phone, string(hash), s.cfg.TermsVersion))
		if err != nil {
			return err
		}
		switch in.Role {
		case "technician":
			_, err = tx.Exec(r.Context(), `INSERT INTO technician_profiles (user_id) VALUES ($1)`, u.ID)
		case "seller":
			_, err = tx.Exec(r.Context(), `INSERT INTO seller_profiles (user_id, shop_name) VALUES ($1, $2)`, u.ID, name)
		}
		return err
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return errStatus(http.StatusConflict, "An account with this email already exists")
	}
	if err != nil {
		return err
	}
	token, err := s.newSession(r, u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, map[string]any{"token": token, "user": u})
	return nil
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) error {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	var id int64
	var hash string
	err := s.db.QueryRow(r.Context(), `SELECT id, password_hash FROM users WHERE email = $1`,
		strings.ToLower(strings.TrimSpace(in.Email))).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil) {
		return errStatus(http.StatusUnauthorized, "Wrong email or password")
	}
	if err != nil {
		return err
	}
	token, err := s.newSession(r, id)
	if err != nil {
		return err
	}
	u, err := scanUser(s.db.QueryRow(r.Context(), `SELECT `+userColumns+` FROM users WHERE id = $1`, id))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "user": u})
	return nil
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request, u *User) error {
	token := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if _, err := s.db.Exec(r.Context(), `DELETE FROM sessions WHERE token_hash = $1`, hashToken(token)); err != nil {
		return err
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, u *User) error {
	writeJSON(w, http.StatusOK, u)
	return nil
}
