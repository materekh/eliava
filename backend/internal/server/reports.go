package server

import (
	"errors"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
)

var reportReasons = []string{"fake", "scam", "unsafe", "spam", "other"}

type Report struct {
	ID           int64     `json:"id"`
	ReporterID   int64     `json:"reporterId"`
	ReporterName string    `json:"reporterName"`
	TargetType   string    `json:"targetType"`
	TargetID     int64     `json:"targetId"`
	TargetLabel  string    `json:"targetLabel"`
	ReportCount  int       `json:"reportCount"`
	Reason       string    `json:"reason"`
	Details      string    `json:"details"`
	CreatedAt    time.Time `json:"createdAt"`
}

// createReport lets any user flag a repair request, a listing, or a technician/seller account for admin review.
func (s *Server) createReport(w http.ResponseWriter, r *http.Request, u *User) error {
	if err := s.limitUser(u, "report", 20, 24*time.Hour); err != nil {
		return err
	}
	var in struct {
		TargetType string `json:"targetType"`
		TargetID   int64  `json:"targetId"`
		Reason     string `json:"reason"`
		Details    string `json:"details"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if !slices.Contains(reportReasons, in.Reason) {
		return badRequest("Choose a reason for the report")
	}
	details := strings.TrimSpace(in.Details)
	if utf8.RuneCountInString(details) > 1000 {
		return badRequest("%s must not exceed %d characters", "Details", 1000)
	}
	var q string
	switch in.TargetType {
	case "post":
		q = `SELECT customer_id FROM repair_posts WHERE id = $1`
	case "listing":
		q = `SELECT seller_id FROM listings WHERE id = $1`
	case "user":
		q = `SELECT id FROM users WHERE id = $1 AND role IN ('technician', 'seller')`
	default:
		return badRequest("Invalid %s", "report target")
	}
	var owner int64
	err := s.db.QueryRow(r.Context(), q, in.TargetID).Scan(&owner)
	if errors.Is(err, pgx.ErrNoRows) {
		return errStatus(http.StatusNotFound, "Not found")
	}
	if err != nil {
		return err
	}
	if owner == u.ID {
		return badRequest("You can't report yourself")
	}
	_, err = s.db.Exec(r.Context(), `
		INSERT INTO reports (reporter_id, target_type, target_id, reason, details) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (reporter_id, target_type, target_id)
		DO UPDATE SET reason = EXCLUDED.reason, details = EXCLUDED.details, status = 'open', created_at = now(), resolved_at = NULL`,
		u.ID, in.TargetType, in.TargetID, in.Reason, details)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, map[string]bool{"ok": true})
	return nil
}

func (s *Server) adminReports(w http.ResponseWriter, r *http.Request, u *User) error {
	rows, err := s.db.Query(r.Context(), `
		SELECT r.id, r.reporter_id, ru.name, r.target_type, r.target_id,
		       COALESCE(CASE r.target_type
		           WHEN 'post' THEN (SELECT title FROM repair_posts WHERE id = r.target_id)
		           WHEN 'listing' THEN (SELECT title FROM listings WHERE id = r.target_id)
		           ELSE (SELECT name || ' (' || role || ', ' || phone || ')' FROM users WHERE id = r.target_id) END, '(deleted)'),
		       (SELECT count(*) FROM reports r2 WHERE r2.target_type = r.target_type AND r2.target_id = r.target_id AND r2.status = 'open')::int,
		       r.reason, r.details, r.created_at
		FROM reports r JOIN users ru ON ru.id = r.reporter_id
		WHERE r.status = 'open'
		ORDER BY r.created_at DESC LIMIT 500`)
	if err != nil {
		return err
	}
	list, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Report, error) {
		var rp Report
		return rp, row.Scan(&rp.ID, &rp.ReporterID, &rp.ReporterName, &rp.TargetType, &rp.TargetID, &rp.TargetLabel,
			&rp.ReportCount, &rp.Reason, &rp.Details, &rp.CreatedAt)
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, list)
	return nil
}

func (s *Server) adminResolveReport(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE reports SET status = 'resolved', resolved_at = now() WHERE id = $1 AND status = 'open'`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errStatus(http.StatusNotFound, "Not found")
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	return nil
}
