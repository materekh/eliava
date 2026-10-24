package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type Category struct {
	ID   int64  `json:"id"`
	Kind string `json:"kind"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) error {
	kind := r.URL.Query().Get("kind")
	rows, err := s.db.Query(r.Context(), `
		SELECT id, kind, slug, name FROM categories WHERE ($1 = '' OR kind = $1) ORDER BY kind, id`, kind)
	if err != nil {
		return err
	}
	cats, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Category, error) {
		var c Category
		return c, row.Scan(&c.ID, &c.Kind, &c.Slug, &c.Name)
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, cats)
	return nil
}

type Technician struct {
	ID              int64      `json:"id"`
	Name            string     `json:"name"`
	Phone           string     `json:"phone"`
	Bio             string     `json:"bio"`
	City            string     `json:"city"`
	Location        *Point     `json:"location"`
	ServiceRadiusKM int        `json:"serviceRadiusKm"`
	Categories      []Category `json:"categories"`
	BoostTetri      int64      `json:"boostTetri"`
	DistanceKM      *float64   `json:"distanceKm,omitempty"`
}

const technicianSelect = `
	SELECT u.id, u.name, u.phone, tp.bio, tp.city,
	       ST_Y(tp.location::geometry), ST_X(tp.location::geometry), tp.service_radius_km,
	       COALESCE((SELECT SUM(b.amount_tetri) FROM boosts b WHERE b.user_id = u.id AND b.ends_at > now()), 0)::bigint AS boost,
	       CASE WHEN $1::float8 IS NULL THEN NULL
	            ELSE ST_Distance(tp.location, ST_SetSRID(ST_MakePoint($2::float8, $1::float8), 4326)::geography) / 1000 END AS distance_km,
	       COALESCE((SELECT json_agg(json_build_object('id', c.id, 'kind', c.kind, 'slug', c.slug, 'name', c.name) ORDER BY c.name)
	                 FROM technician_categories tc JOIN categories c ON c.id = tc.category_id WHERE tc.user_id = u.id), '[]'::json)
	FROM users u JOIN technician_profiles tp ON tp.user_id = u.id`

func scanTechnician(row pgx.Row) (Technician, error) {
	var t Technician
	var lat, lng *float64
	err := row.Scan(&t.ID, &t.Name, &t.Phone, &t.Bio, &t.City, &lat, &lng, &t.ServiceRadiusKM, &t.BoostTetri, &t.DistanceKM, &t.Categories)
	if lat != nil && lng != nil {
		t.Location = &Point{Lat: *lat, Lng: *lng}
	}
	return t, err
}

func (s *Server) searchTechnicians(w http.ResponseWriter, r *http.Request) error {
	p, err := queryPoint(r)
	if err != nil {
		return err
	}
	radius := 25.0
	if v, ok, err := queryFloat(r, "radius_km"); err != nil {
		return err
	} else if ok {
		radius = v
	}
	if radius < 1 || radius > 500 {
		return badRequest("Radius must be between 1 and 500 km")
	}
	var lat, lng *float64
	if p != nil {
		lat, lng = &p.Lat, &p.Lng
	}
	rows, err := s.db.Query(r.Context(), technicianSelect+`
		WHERE u.role = 'technician' AND u.registration_paid_until > now() AND tp.location IS NOT NULL
		  AND ($3 = '' OR EXISTS (SELECT 1 FROM technician_categories tc JOIN categories c ON c.id = tc.category_id
		                          WHERE tc.user_id = u.id AND c.slug = $3))
		  AND ($1::float8 IS NULL OR ST_DWithin(tp.location, ST_SetSRID(ST_MakePoint($2::float8, $1::float8), 4326)::geography, $4 * 1000))
		ORDER BY boost DESC, distance_km ASC NULLS LAST, u.id DESC
		LIMIT 100`, lat, lng, r.URL.Query().Get("category"), radius)
	if err != nil {
		return err
	}
	techs, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Technician, error) { return scanTechnician(row) })
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, techs)
	return nil
}

func (s *Server) getTechnician(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	t, err := scanTechnician(s.db.QueryRow(r.Context(), technicianSelect+`
		WHERE u.id = $3 AND u.role = 'technician' AND u.registration_paid_until > now()`, nil, nil, id))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, t)
	return nil
}

func (s *Server) getTechnicianProfile(w http.ResponseWriter, r *http.Request, u *User) error {
	t, err := scanTechnician(s.db.QueryRow(r.Context(), technicianSelect+` WHERE u.id = $3`, nil, nil, u.ID))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, t)
	return nil
}

func (s *Server) putTechnicianProfile(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		Bio             string  `json:"bio"`
		City            string  `json:"city"`
		Location        *Point  `json:"location"`
		ServiceRadiusKM int     `json:"serviceRadiusKm"`
		CategoryIDs     []int64 `json:"categoryIds"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	bio, err := trimmed(in.Bio, "Bio", 0, 2000)
	if err != nil {
		return err
	}
	city, err := trimmed(in.City, "City", 0, 100)
	if err != nil {
		return err
	}
	if err := in.Location.validate(); err != nil {
		return err
	}
	if in.ServiceRadiusKM < 1 || in.ServiceRadiusKM > 500 {
		return badRequest("Service radius must be between 1 and 500 km")
	}
	if len(in.CategoryIDs) == 0 || len(in.CategoryIDs) > 20 {
		return badRequest("Choose between 1 and 20 specialities")
	}
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var valid int
		if err := tx.QueryRow(r.Context(), `SELECT count(*) FROM categories WHERE kind = 'repair' AND id = ANY($1)`, in.CategoryIDs).Scan(&valid); err != nil {
			return err
		}
		if valid != len(uniqueIDs(in.CategoryIDs)) {
			return badRequest("Unknown speciality")
		}
		if _, err := tx.Exec(r.Context(), `
			INSERT INTO technician_profiles (user_id, bio, city, location, service_radius_km, updated_at)
			VALUES ($1, $2, $3, ST_SetSRID(ST_MakePoint($4, $5), 4326)::geography, $6, $7)
			ON CONFLICT (user_id) DO UPDATE SET bio = EXCLUDED.bio, city = EXCLUDED.city, location = EXCLUDED.location,
			    service_radius_km = EXCLUDED.service_radius_km, updated_at = EXCLUDED.updated_at`,
			u.ID, bio, city, in.Location.Lng, in.Location.Lat, in.ServiceRadiusKM, time.Now()); err != nil {
			return err
		}
		if _, err := tx.Exec(r.Context(), `DELETE FROM technician_categories WHERE user_id = $1`, u.ID); err != nil {
			return err
		}
		_, err := tx.Exec(r.Context(), `
			INSERT INTO technician_categories (user_id, category_id) SELECT $1, unnest($2::bigint[]) ON CONFLICT DO NOTHING`,
			u.ID, uniqueIDs(in.CategoryIDs))
		return err
	})
	if err != nil {
		return err
	}
	return s.getTechnicianProfile(w, r, u)
}

func uniqueIDs(ids []int64) []int64 {
	seen := map[int64]bool{}
	var out []int64
	for _, id := range ids {
		if !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}

func parseID(v, field string) (int64, error) {
	id, err := strconv.ParseInt(v, 10, 64)
	if err != nil || id <= 0 {
		return 0, badRequest("Invalid %s", field)
	}
	return id, nil
}
