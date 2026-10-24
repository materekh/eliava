package server

import (
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
)

type RevealLog struct {
	ID              int64     `json:"id"`
	TechnicianID    int64     `json:"technicianId"`
	TechnicianName  string    `json:"technicianName"`
	TechnicianPhone string    `json:"technicianPhone"`
	TechnicianEmail string    `json:"technicianEmail"`
	PostID          int64     `json:"postId"`
	PostTitle       string    `json:"postTitle"`
	CustomerName    string    `json:"customerName"`
	PriceTetri      int64     `json:"priceTetri"`
	CreatedAt       time.Time `json:"createdAt"`
}

func (s *Server) adminReveals(w http.ResponseWriter, r *http.Request, u *User) error {
	rows, err := s.db.Query(r.Context(), `
		SELECT cr.id, t.id, t.name, t.phone, t.email, p.id, p.title, cu.name, cr.price_tetri, cr.created_at
		FROM contact_reveals cr
		JOIN users t ON t.id = cr.technician_id
		JOIN repair_posts p ON p.id = cr.post_id
		JOIN users cu ON cu.id = p.customer_id
		ORDER BY cr.created_at DESC LIMIT 500`)
	if err != nil {
		return err
	}
	logs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (RevealLog, error) {
		var l RevealLog
		return l, row.Scan(&l.ID, &l.TechnicianID, &l.TechnicianName, &l.TechnicianPhone, &l.TechnicianEmail,
			&l.PostID, &l.PostTitle, &l.CustomerName, &l.PriceTetri, &l.CreatedAt)
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, logs)
	return nil
}

func (s *Server) adminFlaggedPosts(w http.ResponseWriter, r *http.Request, u *User) error {
	posts, err := s.queryPosts(r.Context(), u, "WHERE p.flagged ORDER BY p.created_at DESC LIMIT 200", u.ID, nil, nil)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, posts)
	return nil
}

type stallInput struct {
	Code     *string `json:"code"`
	Name     *string `json:"name"`
	Location *Point  `json:"location"`
}

func (s *Server) adminCreateStall(w http.ResponseWriter, r *http.Request, u *User) error {
	var in stallInput
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if in.Code == nil {
		return badRequest("Code is required")
	}
	code, err := trimmed(*in.Code, "Code", 1, 20)
	if err != nil {
		return err
	}
	name := ""
	if in.Name != nil {
		name = *in.Name
	}
	if err := in.Location.validate(); err != nil {
		return err
	}
	_, err = s.db.Exec(r.Context(), `INSERT INTO stalls (code, name, location) VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326)::geography)`,
		code, name, in.Location.Lng, in.Location.Lat)
	if err != nil {
		return errStatus(http.StatusConflict, "A stall with this code already exists")
	}
	return s.listStalls(w, r)
}

func (s *Server) adminUpdateStall(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in stallInput
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	var lat, lng *float64
	if in.Location != nil {
		if err := in.Location.validate(); err != nil {
			return err
		}
		lat, lng = &in.Location.Lat, &in.Location.Lng
	}
	tag, err := s.db.Exec(r.Context(), `
		UPDATE stalls SET code = COALESCE($2, code), name = COALESCE($3, name),
		       location = CASE WHEN $4::float8 IS NULL THEN location ELSE ST_SetSRID(ST_MakePoint($5::float8, $4::float8), 4326)::geography END
		WHERE id = $1`, id, in.Code, in.Name, lat, lng)
	if err != nil {
		return errStatus(http.StatusConflict, "A stall with this code already exists")
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return s.listStalls(w, r)
}
