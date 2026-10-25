package server

import (
	"context"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Contact struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
}

type Post struct {
	ID          int64     `json:"id"`
	CustomerID  int64     `json:"customerId"`
	Category    Category  `json:"category"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Address     string    `json:"address"`
	Location    Point     `json:"location"`
	HasParts    bool      `json:"hasParts"`
	Status      string    `json:"status"`
	Estimate    Estimate  `json:"estimate"`
	Flagged     bool      `json:"flagged"`
	FlagReason  string    `json:"flagReason,omitempty"`
	DuplicateOf *int64    `json:"duplicateOf,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	Media       []Media   `json:"media"`
	DistanceKM  *float64  `json:"distanceKm,omitempty"`
	Revealed    bool      `json:"revealed"`
	ViewCount   *int      `json:"viewCount,omitempty"`
	Contact     *Contact  `json:"contact,omitempty"`
}

// postSelect expects $1 = viewer user id, $2/$3 = optional lat/lng for distance.
const postSelect = `
	SELECT p.id, p.customer_id, c.id, c.kind, c.slug, c.name, p.title, p.description, p.address,
	       ST_Y(p.location::geometry), ST_X(p.location::geometry), p.has_parts, p.status,
	       p.estimate_min_tetri, p.estimate_max_tetri, p.estimate_note, p.flagged, p.flag_reason, p.duplicate_of, p.created_at,
	       CASE WHEN $2::float8 IS NULL THEN NULL
	            ELSE ST_Distance(p.location, ST_SetSRID(ST_MakePoint($3::float8, $2::float8), 4326)::geography) / 1000 END AS distance_km,
	       EXISTS (SELECT 1 FROM contact_reveals cr WHERE cr.post_id = p.id AND cr.technician_id = $1),
	       (SELECT count(*) FROM contact_reveals cr WHERE cr.post_id = p.id)::int,
	       cu.name, cu.phone
	FROM repair_posts p
	JOIN categories c ON c.id = p.category_id
	JOIN users cu ON cu.id = p.customer_id`

func scanPost(row pgx.Row, viewer *User) (Post, error) {
	var p Post
	var views int
	var contact Contact
	err := row.Scan(&p.ID, &p.CustomerID, &p.Category.ID, &p.Category.Kind, &p.Category.Slug, &p.Category.Name,
		&p.Title, &p.Description, &p.Address, &p.Location.Lat, &p.Location.Lng, &p.HasParts, &p.Status,
		&p.Estimate.MinTetri, &p.Estimate.MaxTetri, &p.Estimate.Note, &p.Flagged, &p.FlagReason, &p.DuplicateOf, &p.CreatedAt,
		&p.DistanceKM, &p.Revealed, &views, &contact.Name, &contact.Phone)
	if err != nil {
		return p, err
	}
	owner := viewer.ID == p.CustomerID || viewer.Role == "admin"
	if owner {
		p.ViewCount = &views
	} else {
		// Technicians must not see why or whether a post was flagged.
		p.Flagged, p.FlagReason, p.DuplicateOf = false, "", nil
	}
	if owner || p.Revealed {
		p.Contact = &contact
	} else {
		// Until a technician views the number, show only the rough area (~1 km), never the exact home.
		p.Address = ""
		p.Location.Lat = math.Round(p.Location.Lat*100) / 100
		p.Location.Lng = math.Round(p.Location.Lng*100) / 100
		if p.DistanceKM != nil {
			d := math.Max(1, math.Ceil(*p.DistanceKM))
			p.DistanceKM = &d
		}
	}
	return p, nil
}

func (s *Server) queryPosts(ctx context.Context, viewer *User, where string, args ...any) ([]Post, error) {
	rows, err := s.db.Query(ctx, postSelect+" "+where, args...)
	if err != nil {
		return nil, err
	}
	posts, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Post, error) { return scanPost(row, viewer) })
	if err != nil {
		return nil, err
	}
	return posts, s.attachPostMedia(ctx, posts)
}

func (s *Server) attachPostMedia(ctx context.Context, posts []Post) error {
	if len(posts) == 0 {
		return nil
	}
	ids := make([]int64, len(posts))
	index := map[int64]int{}
	for i := range posts {
		ids[i] = posts[i].ID
		index[posts[i].ID] = i
		posts[i].Media = []Media{}
	}
	rows, err := s.db.Query(ctx, `SELECT post_id, id, kind, path FROM media WHERE post_id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var postID int64
		var m Media
		var rel string
		if err := rows.Scan(&postID, &m.ID, &m.Kind, &rel); err != nil {
			return err
		}
		m.URL = s.mediaURL(rel)
		posts[index[postID]].Media = append(posts[index[postID]].Media, m)
	}
	return rows.Err()
}

func (s *Server) createPost(w http.ResponseWriter, r *http.Request, u *User) error {
	if err := s.limitUser(u, "post", 10, 24*time.Hour); err != nil {
		return err
	}
	if err := parseMultipart(w, r); err != nil {
		return err
	}
	f := r.FormValue
	categoryID, err := parseID(f("categoryId"), "category")
	if err != nil {
		return err
	}
	title, err := trimmed(f("title"), "Title", 5, 120)
	if err != nil {
		return err
	}
	description, err := trimmed(f("description"), "Description", 20, 4000)
	if err != nil {
		return err
	}
	address, err := trimmed(f("address"), "Address", 0, 300)
	if err != nil {
		return err
	}
	lat, errLat := strconv.ParseFloat(f("lat"), 64)
	lng, errLng := strconv.ParseFloat(f("lng"), 64)
	if errLat != nil || errLng != nil {
		return badRequest("Pick your location on the map")
	}
	loc := Point{Lat: lat, Lng: lng}
	if err := loc.validate(); err != nil {
		return err
	}
	hasParts := f("hasParts") == "true"

	est, err := s.estimate(r.Context(), categoryID, description, hasParts, loc)
	if err != nil {
		return err
	}
	dup, err := s.findDuplicate(r.Context(), u.ID, categoryID, title+" "+description)
	if err != nil {
		return err
	}

	files, err := s.saveUploads(r, "media", "posts")
	if err != nil {
		return err
	}
	hasImage := false
	for _, file := range files {
		hasImage = hasImage || file.kind == "image"
	}
	if !hasImage {
		s.removeFiles(files)
		return badRequest("Add at least one photo of the problem")
	}

	var id int64
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var dupID *int64
		flagReason := ""
		if dup != nil {
			dupID, flagReason = &dup.postID, dup.reason
		}
		err := tx.QueryRow(r.Context(), `
			INSERT INTO repair_posts (customer_id, category_id, title, description, address, location, has_parts,
			                          estimate_min_tetri, estimate_max_tetri, estimate_note, flagged, flag_reason, duplicate_of)
			VALUES ($1, $2, $3, $4, $5, ST_SetSRID(ST_MakePoint($6, $7), 4326)::geography, $8, $9, $10, $11, $12, $13, $14)
			RETURNING id`,
			u.ID, categoryID, title, description, address, loc.Lng, loc.Lat, hasParts,
			est.MinTetri, est.MaxTetri, est.Note, dup != nil, flagReason, dupID).Scan(&id)
		if err != nil {
			return err
		}
		return insertMedia(r.Context(), tx, u.ID, "post_id", id, files)
	})
	if err != nil {
		s.removeFiles(files)
		return err
	}
	posts, err := s.queryPosts(r.Context(), u, "WHERE p.id = $4", u.ID, nil, nil, id)
	if err != nil {
		return err
	}
	posts[0].Estimate.Source = est.Source
	writeJSON(w, http.StatusCreated, posts[0])
	return nil
}

func insertMedia(ctx context.Context, tx pgx.Tx, ownerID int64, column string, parentID int64, files []savedFile) error {
	for _, file := range files {
		if _, err := tx.Exec(ctx, `INSERT INTO media (owner_id, `+column+`, kind, path, mime, size_bytes) VALUES ($1, $2, $3, $4, $5, $6)`,
			ownerID, parentID, file.kind, file.rel, file.mime, file.size); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) myPosts(w http.ResponseWriter, r *http.Request, u *User) error {
	posts, err := s.queryPosts(r.Context(), u, "WHERE p.customer_id = $1 ORDER BY p.created_at DESC LIMIT 100", u.ID, nil, nil)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, posts)
	return nil
}

// nearbyPosts lists open, unflagged posts around the technician (their profile location unless lat/lng are given).
func (s *Server) nearbyPosts(w http.ResponseWriter, r *http.Request, u *User) error {
	p, err := queryPoint(r)
	if err != nil {
		return err
	}
	var radius float64
	var profLat, profLng *float64
	_ = s.db.QueryRow(r.Context(), `
		SELECT ST_Y(location::geometry), ST_X(location::geometry), service_radius_km FROM technician_profiles WHERE user_id = $1`,
		u.ID).Scan(&profLat, &profLng, &radius)
	if p == nil {
		if profLat == nil {
			return badRequest("Set your location in your technician profile first")
		}
		p = &Point{Lat: *profLat, Lng: *profLng}
	}
	if v, ok, err := queryFloat(r, "radius_km"); err != nil {
		return err
	} else if ok {
		radius = v
	}
	if radius < 1 || radius > 500 {
		radius = 25
	}
	category := r.URL.Query().Get("category")
	mineOnly := r.URL.Query().Get("mine") == "1"
	posts, err := s.queryPosts(r.Context(), u, `
		WHERE p.status = 'open' AND NOT p.flagged AND p.created_at > now() - interval '60 days'
		  AND ST_DWithin(p.location, ST_SetSRID(ST_MakePoint($3::float8, $2::float8), 4326)::geography, $4 * 1000)
		  AND ($5 = '' OR c.slug = $5)
		  AND (NOT $6 OR p.category_id IN (SELECT category_id FROM technician_categories WHERE user_id = $1))
		ORDER BY distance_km ASC, p.created_at DESC
		LIMIT 100`, u.ID, p.Lat, p.Lng, radius, category, mineOnly)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, posts)
	return nil
}

func (s *Server) loadPost(ctx context.Context, u *User, id int64) (Post, error) {
	posts, err := s.queryPosts(ctx, u, "WHERE p.id = $4", u.ID, nil, nil, id)
	if err != nil {
		return Post{}, err
	}
	if len(posts) == 0 {
		return Post{}, pgx.ErrNoRows
	}
	return posts[0], nil
}

func (s *Server) getPost(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	p, err := s.loadPost(r.Context(), u, id)
	if err != nil {
		return err
	}
	if u.Role == "customer" && p.CustomerID != u.ID {
		return errStatus(http.StatusNotFound, "Not found")
	}
	writeJSON(w, http.StatusOK, p)
	return nil
}

// revealContact charges the technician once per post and records who viewed the customer's number.
func (s *Server) revealContact(w http.ResponseWriter, r *http.Request, u *User) error {
	if err := s.limitUser(u, "reveal", 60, time.Hour); err != nil {
		return err
	}
	id, err := pathID(r)
	if err != nil {
		return err
	}
	if !u.RegistrationActive {
		return errStatus(http.StatusPaymentRequired, "Pay the technician registration fee to view customer numbers")
	}
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var status string
		if err := tx.QueryRow(r.Context(), `SELECT status FROM repair_posts WHERE id = $1`, id).Scan(&status); err != nil {
			return err
		}
		var revealID int64
		err := tx.QueryRow(r.Context(), `
			INSERT INTO contact_reveals (technician_id, post_id, price_tetri) VALUES ($1, $2, $3)
			ON CONFLICT (technician_id, post_id) DO NOTHING RETURNING id`, u.ID, id, s.cfg.RevealPrice).Scan(&revealID)
		if err == pgx.ErrNoRows {
			return nil // already revealed, no new charge
		}
		if err != nil {
			return err
		}
		if status != "open" {
			return errStatus(http.StatusConflict, "This request is closed")
		}
		return charge(r.Context(), tx, u.ID, s.cfg.RevealPrice, "View customer number", "post:"+strconv.FormatInt(id, 10))
	})
	if err != nil {
		return err
	}
	p, err := s.loadPost(r.Context(), u, id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, p)
	return nil
}

func (s *Server) closePost(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	tag, err := s.db.Exec(r.Context(), `UPDATE repair_posts SET status = 'closed' WHERE id = $1 AND (customer_id = $2 OR $3)`,
		id, u.ID, u.Role == "admin")
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

type Viewer struct {
	TechnicianID int64     `json:"technicianId"`
	Name         string    `json:"name"`
	ViewedAt     time.Time `json:"viewedAt"`
}

func (s *Server) postViewers(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(r.Context(), `
		SELECT t.id, t.name, cr.created_at
		FROM contact_reveals cr JOIN repair_posts p ON p.id = cr.post_id JOIN users t ON t.id = cr.technician_id
		WHERE cr.post_id = $1 AND (p.customer_id = $2 OR $3)
		ORDER BY cr.created_at DESC`, id, u.ID, u.Role == "admin")
	if err != nil {
		return err
	}
	viewers, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Viewer, error) {
		var v Viewer
		return v, row.Scan(&v.TechnicianID, &v.Name, &v.ViewedAt)
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, viewers)
	return nil
}

func parseGEL(v string) (int64, error) {
	v = strings.TrimSpace(strings.ReplaceAll(v, ",", "."))
	f, err := strconv.ParseFloat(v, 64)
	if err != nil || f <= 0 || f > 1_000_000 {
		return 0, badRequest("Enter a valid price in GEL")
	}
	return int64(f*100 + 0.5), nil
}
