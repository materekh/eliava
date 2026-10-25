package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

// BootstrapAdmin makes email an admin whose password is password (from ADMIN_PASSWORD).
// If someone else registered that email first, the password is reset and their sessions are ended,
// so only whoever controls the server's environment can hold the admin account.
func BootstrapAdmin(ctx context.Context, db *pgxpool.Pool, email, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || password == "" {
		return nil
	}
	if len(password) < 12 || len(password) > 72 {
		return errors.New("ADMIN_PASSWORD must be 12-72 characters")
	}
	var id int64
	var hash string
	err := db.QueryRow(ctx, `SELECT id, password_hash FROM users WHERE email = $1`, email).Scan(&id, &hash)
	if errors.Is(err, pgx.ErrNoRows) {
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		_, err = db.Exec(ctx, `INSERT INTO users (role, name, email, phone, password_hash, terms_version)
			VALUES ('admin', 'Admin', $1, '', $2, 'admin')`, email, string(h))
		return err
	}
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil {
		h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err := db.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, string(h)); err != nil {
			return err
		}
		if _, err := db.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, id); err != nil {
			return err
		}
	}
	_, err = db.Exec(ctx, `UPDATE users SET role = 'admin' WHERE id = $1`, id)
	return err
}

// deleteAccount anonymises the user and removes their photos, profiles and open requests.
// Orders, payments and number-view records are kept (anonymised) for accounting and disputes.
func (s *Server) deleteAccount(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if u.Role == "admin" {
		return badRequest("Admin accounts can't be deleted here")
	}
	var hash string
	if err := s.db.QueryRow(r.Context(), `SELECT password_hash FROM users WHERE id = $1`, u.ID).Scan(&hash); err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(in.Password)) != nil {
		return errStatus(http.StatusForbidden, "Wrong password")
	}
	var files []savedFile
	rows, err := s.db.Query(r.Context(), `SELECT path FROM media WHERE owner_id = $1`, u.ID)
	if err != nil {
		return err
	}
	for rows.Next() {
		var f savedFile
		if err := rows.Scan(&f.rel); err != nil {
			rows.Close()
			return err
		}
		files = append(files, f)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		for _, q := range []string{
			`UPDATE repair_posts SET status = 'closed' WHERE customer_id = $1 AND status = 'open'`,
			`DELETE FROM media WHERE owner_id = $1`,
			`UPDATE listings SET active = false WHERE seller_id = $1`,
			`DELETE FROM technician_categories WHERE user_id = $1`,
			`DELETE FROM technician_profiles WHERE user_id = $1`,
			`DELETE FROM sessions WHERE user_id = $1`,
			`UPDATE users SET name = 'Deleted user', email = 'deleted-' || id || '@deleted.invalid', phone = '', password_hash = '!' WHERE id = $1`,
		} {
			if _, err := tx.Exec(r.Context(), q, u.ID); err != nil {
				return fmt.Errorf("delete account: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	s.removeFiles(files)
	w.WriteHeader(http.StatusNoContent)
	return nil
}
