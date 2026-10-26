package server

import (
	"net/http"
	"time"
)

type categoryStat struct {
	Slug        string `json:"slug"`
	Technicians int    `json:"technicians"`
}

// categoryStats counts registered technicians per repair category for the home page tiles.
func (s *Server) categoryStats(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.db.Query(r.Context(), `
		SELECT c.slug, COUNT(u.id)::int
		FROM categories c
		LEFT JOIN technician_categories tc ON tc.category_id = c.id
		LEFT JOIN users u ON u.id = tc.user_id AND u.role = 'technician' AND u.registration_paid_until > now()
		WHERE c.kind = 'repair'
		GROUP BY c.id, c.slug ORDER BY c.id`)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []categoryStat{}
	for rows.Next() {
		var st categoryStat
		if err := rows.Scan(&st.Slug, &st.Technicians); err != nil {
			return err
		}
		out = append(out, st)
	}
	writeJSON(w, http.StatusOK, out)
	return rows.Err()
}

type recentPost struct {
	ID        int64     `json:"id"`
	Title     string    `json:"title"`
	Category  string    `json:"category"`
	Area      string    `json:"area"`
	CreatedAt time.Time `json:"createdAt"`
	Views     int       `json:"views"`
	MinTetri  int64     `json:"minTetri"`
	MaxTetri  int64     `json:"maxTetri"`
}

// recentPosts is the public feed of open requests: no description, address, exact location, name or phone.
func (s *Server) recentPosts(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.db.Query(r.Context(), `
		SELECT p.id, p.title, c.slug, ST_Y(p.location::geometry), ST_X(p.location::geometry), p.created_at,
		       (SELECT COUNT(*) FROM contact_reveals cr WHERE cr.post_id = p.id)::int, p.estimate_min_tetri, p.estimate_max_tetri
		FROM repair_posts p JOIN categories c ON c.id = p.category_id
		WHERE p.status = 'open' AND NOT p.flagged
		ORDER BY p.created_at DESC LIMIT 6`)
	if err != nil {
		return err
	}
	defer rows.Close()
	out := []recentPost{}
	for rows.Next() {
		var p recentPost
		var pt Point
		if err := rows.Scan(&p.ID, &p.Title, &p.Category, &pt.Lat, &pt.Lng, &p.CreatedAt, &p.Views, &p.MinTetri, &p.MaxTetri); err != nil {
			return err
		}
		p.Area, _ = areaFor(pt)
		out = append(out, p)
	}
	writeJSON(w, http.StatusOK, out)
	return rows.Err()
}
