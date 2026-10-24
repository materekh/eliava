package server

import (
	"context"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
)

type Stall struct {
	ID          int64  `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Location    Point  `json:"location"`
	SellerCount int    `json:"sellerCount"`
}

func (s *Server) listStalls(w http.ResponseWriter, r *http.Request) error {
	rows, err := s.db.Query(r.Context(), `
		SELECT st.id, st.code, st.name, ST_Y(st.location::geometry), ST_X(st.location::geometry),
		       (SELECT count(*) FROM seller_profiles sp JOIN users u ON u.id = sp.user_id
		        WHERE sp.stall_id = st.id AND u.registration_paid_until > now())::int
		FROM stalls st ORDER BY st.code`)
	if err != nil {
		return err
	}
	stalls, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Stall, error) {
		var st Stall
		return st, row.Scan(&st.ID, &st.Code, &st.Name, &st.Location.Lat, &st.Location.Lng, &st.SellerCount)
	})
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, stalls)
	return nil
}

type SellerProfile struct {
	UserID    int64   `json:"userId"`
	Name      string  `json:"name"`
	Phone     string  `json:"phone"`
	ShopName  string  `json:"shopName"`
	StallID   *int64  `json:"stallId"`
	StallCode *string `json:"stallCode"`
	Address   string  `json:"address"`
	Location  *Point  `json:"location"`
}

func (s *Server) loadSellerProfile(ctx context.Context, userID int64) (SellerProfile, error) {
	var sp SellerProfile
	var lat, lng *float64
	err := s.db.QueryRow(ctx, `
		SELECT u.id, u.name, u.phone, sp.shop_name, sp.stall_id, st.code, sp.address,
		       ST_Y(COALESCE(sp.location, st.location)::geometry), ST_X(COALESCE(sp.location, st.location)::geometry)
		FROM users u JOIN seller_profiles sp ON sp.user_id = u.id LEFT JOIN stalls st ON st.id = sp.stall_id
		WHERE u.id = $1`, userID).Scan(&sp.UserID, &sp.Name, &sp.Phone, &sp.ShopName, &sp.StallID, &sp.StallCode, &sp.Address, &lat, &lng)
	if lat != nil && lng != nil {
		sp.Location = &Point{Lat: *lat, Lng: *lng}
	}
	return sp, err
}

func (s *Server) getSellerProfile(w http.ResponseWriter, r *http.Request, u *User) error {
	sp, err := s.loadSellerProfile(r.Context(), u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, sp)
	return nil
}

func (s *Server) putSellerProfile(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		ShopName string `json:"shopName"`
		StallID  *int64 `json:"stallId"`
		Address  string `json:"address"`
		Location *Point `json:"location"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	shop, err := trimmed(in.ShopName, "Shop name", 2, 100)
	if err != nil {
		return err
	}
	address, err := trimmed(in.Address, "Address", 0, 300)
	if err != nil {
		return err
	}
	if in.StallID == nil && in.Location == nil {
		return badRequest("Choose your Eliava stall or pin your shop on the map")
	}
	var lat, lng *float64
	if in.Location != nil {
		if err := in.Location.validate(); err != nil {
			return err
		}
		lat, lng = &in.Location.Lat, &in.Location.Lng
	}
	_, err = s.db.Exec(r.Context(), `
		INSERT INTO seller_profiles (user_id, shop_name, stall_id, address, location, updated_at)
		VALUES ($1, $2, $3, $4, CASE WHEN $5::float8 IS NULL THEN NULL ELSE ST_SetSRID(ST_MakePoint($6::float8, $5::float8), 4326)::geography END, now())
		ON CONFLICT (user_id) DO UPDATE SET shop_name = EXCLUDED.shop_name, stall_id = EXCLUDED.stall_id,
		    address = EXCLUDED.address, location = EXCLUDED.location, updated_at = EXCLUDED.updated_at`,
		u.ID, shop, in.StallID, address, lat, lng)
	if err != nil {
		return badRequest("Unknown stall")
	}
	return s.getSellerProfile(w, r, u)
}

type Listing struct {
	ID          int64         `json:"id"`
	Category    Category      `json:"category"`
	Title       string        `json:"title"`
	Description string        `json:"description"`
	PriceTetri  int64         `json:"priceTetri"`
	Stock       int           `json:"stock"`
	Active      bool          `json:"active"`
	CreatedAt   time.Time     `json:"createdAt"`
	Seller      SellerProfile `json:"seller"`
	BoostTetri  int64         `json:"boostTetri"`
	Media       []Media       `json:"media"`
}

const listingSelect = `
	SELECT l.id, c.id, c.kind, c.slug, c.name, l.title, l.description, l.price_tetri, l.stock, l.active, l.created_at,
	       u.id, u.name, u.phone, sp.shop_name, sp.stall_id, st.code, sp.address,
	       ST_Y(COALESCE(sp.location, st.location)::geometry), ST_X(COALESCE(sp.location, st.location)::geometry),
	       COALESCE((SELECT SUM(b.amount_tetri) FROM boosts b WHERE b.user_id = u.id AND b.ends_at > now()), 0)::bigint AS boost
	FROM listings l
	JOIN categories c ON c.id = l.category_id
	JOIN users u ON u.id = l.seller_id
	JOIN seller_profiles sp ON sp.user_id = u.id
	LEFT JOIN stalls st ON st.id = sp.stall_id`

func (s *Server) queryListings(ctx context.Context, where string, args ...any) ([]Listing, error) {
	rows, err := s.db.Query(ctx, listingSelect+" "+where, args...)
	if err != nil {
		return nil, err
	}
	listings, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (Listing, error) {
		var l Listing
		var lat, lng *float64
		err := row.Scan(&l.ID, &l.Category.ID, &l.Category.Kind, &l.Category.Slug, &l.Category.Name, &l.Title, &l.Description,
			&l.PriceTetri, &l.Stock, &l.Active, &l.CreatedAt,
			&l.Seller.UserID, &l.Seller.Name, &l.Seller.Phone, &l.Seller.ShopName, &l.Seller.StallID, &l.Seller.StallCode,
			&l.Seller.Address, &lat, &lng, &l.BoostTetri)
		if lat != nil && lng != nil {
			l.Seller.Location = &Point{Lat: *lat, Lng: *lng}
		}
		l.Media = []Media{}
		return l, err
	})
	if err != nil || len(listings) == 0 {
		return listings, err
	}
	ids := make([]int64, len(listings))
	index := map[int64]int{}
	for i, l := range listings {
		ids[i] = l.ID
		index[l.ID] = i
	}
	mrows, err := s.db.Query(ctx, `SELECT listing_id, id, kind, path FROM media WHERE listing_id = ANY($1) ORDER BY id`, ids)
	if err != nil {
		return nil, err
	}
	defer mrows.Close()
	for mrows.Next() {
		var lid int64
		var m Media
		var rel string
		if err := mrows.Scan(&lid, &m.ID, &m.Kind, &rel); err != nil {
			return nil, err
		}
		m.URL = s.mediaURL(rel)
		listings[index[lid]].Media = append(listings[index[lid]].Media, m)
	}
	return listings, mrows.Err()
}

func (s *Server) searchListings(w http.ResponseWriter, r *http.Request) error {
	q := r.URL.Query()
	var stallID, sellerID int64
	var err error
	if v := q.Get("stall"); v != "" {
		if stallID, err = parseID(v, "stall"); err != nil {
			return err
		}
	}
	if v := q.Get("seller"); v != "" {
		if sellerID, err = parseID(v, "seller"); err != nil {
			return err
		}
	}
	listings, err := s.queryListings(r.Context(), `
		WHERE l.active AND l.stock > 0 AND u.registration_paid_until > now()
		  AND ($1 = '' OR c.slug = $1)
		  AND ($2 = 0 OR sp.stall_id = $2)
		  AND ($3 = 0 OR u.id = $3)
		  AND ($4 = '' OR l.title ILIKE '%' || $4 || '%' OR l.description ILIKE '%' || $4 || '%' OR similarity(l.title, $4) > 0.3)
		ORDER BY boost DESC, l.created_at DESC
		LIMIT 100`, q.Get("category"), stallID, sellerID, q.Get("q"))
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, listings)
	return nil
}

func (s *Server) getListing(w http.ResponseWriter, r *http.Request) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	listings, err := s.queryListings(r.Context(), "WHERE l.id = $1", id)
	if err != nil {
		return err
	}
	if len(listings) == 0 {
		return pgx.ErrNoRows
	}
	writeJSON(w, http.StatusOK, listings[0])
	return nil
}

func (s *Server) myListings(w http.ResponseWriter, r *http.Request, u *User) error {
	listings, err := s.queryListings(r.Context(), "WHERE l.seller_id = $1 ORDER BY l.created_at DESC", u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, listings)
	return nil
}

func (s *Server) createListing(w http.ResponseWriter, r *http.Request, u *User) error {
	if !u.RegistrationActive {
		return errStatus(http.StatusPaymentRequired, "Pay the seller registration fee to post listings")
	}
	if err := parseMultipart(w, r); err != nil {
		return err
	}
	f := r.FormValue
	categoryID, err := parseID(f("categoryId"), "category")
	if err != nil {
		return err
	}
	var kind string
	if err := s.db.QueryRow(r.Context(), `SELECT kind FROM categories WHERE id = $1`, categoryID).Scan(&kind); err != nil || kind != "part" {
		return badRequest("Choose a valid market category")
	}
	title, err := trimmed(f("title"), "Title", 3, 120)
	if err != nil {
		return err
	}
	description, err := trimmed(f("description"), "Description", 0, 4000)
	if err != nil {
		return err
	}
	price, err := parseGEL(f("price"))
	if err != nil {
		return err
	}
	stock, err := strconv.Atoi(f("stock"))
	if err != nil || stock < 0 || stock > 100000 {
		return badRequest("Enter how many you have in stock")
	}
	files, err := s.saveUploads(r, "media", "listings")
	if err != nil {
		return err
	}
	var id int64
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		if err := tx.QueryRow(r.Context(), `
			INSERT INTO listings (seller_id, category_id, title, description, price_tetri, stock)
			VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`, u.ID, categoryID, title, description, price, stock).Scan(&id); err != nil {
			return err
		}
		return insertMedia(r.Context(), tx, u.ID, "listing_id", id, files)
	})
	if err != nil {
		s.removeFiles(files)
		return err
	}
	listings, err := s.queryListings(r.Context(), "WHERE l.id = $1", id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, listings[0])
	return nil
}

func (s *Server) updateListing(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in struct {
		PriceTetri *int64 `json:"priceTetri"`
		Stock      *int   `json:"stock"`
		Active     *bool  `json:"active"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if in.PriceTetri != nil && (*in.PriceTetri <= 0 || *in.PriceTetri > 100_000_000) {
		return badRequest("Invalid price")
	}
	if in.Stock != nil && (*in.Stock < 0 || *in.Stock > 100000) {
		return badRequest("Invalid stock")
	}
	tag, err := s.db.Exec(r.Context(), `
		UPDATE listings SET price_tetri = COALESCE($3, price_tetri), stock = COALESCE($4, stock), active = COALESCE($5, active)
		WHERE id = $1 AND (seller_id = $2 OR $6)`, id, u.ID, in.PriceTetri, in.Stock, in.Active, u.Role == "admin")
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	listings, err := s.queryListings(r.Context(), "WHERE l.id = $1", id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, listings[0])
	return nil
}

type Order struct {
	ID               int64     `json:"id"`
	ListingID        int64     `json:"listingId"`
	ListingTitle     string    `json:"listingTitle"`
	CustomerName     string    `json:"customerName"`
	CustomerPhone    string    `json:"customerPhone"`
	SellerName       string    `json:"sellerName"`
	SellerPhone      string    `json:"sellerPhone"`
	StallCode        *string   `json:"stallCode"`
	Quantity         int       `json:"quantity"`
	UnitPriceTetri   int64     `json:"unitPriceTetri"`
	Delivery         bool      `json:"delivery"`
	DeliveryAddress  string    `json:"deliveryAddress"`
	DeliveryFeeTetri int64     `json:"deliveryFeeTetri"`
	TotalTetri       int64     `json:"totalTetri"`
	Status           string    `json:"status"`
	CreatedAt        time.Time `json:"createdAt"`
}

func (s *Server) queryOrders(ctx context.Context, where string, args ...any) ([]Order, error) {
	rows, err := s.db.Query(ctx, `
		SELECT o.id, o.listing_id, l.title, cu.name, cu.phone, se.name, se.phone, st.code, o.quantity, o.unit_price_tetri,
		       o.delivery, o.delivery_address, o.delivery_fee_tetri, o.total_tetri, o.status, o.created_at
		FROM orders o
		JOIN listings l ON l.id = o.listing_id
		JOIN users cu ON cu.id = o.customer_id
		JOIN users se ON se.id = o.seller_id
		LEFT JOIN seller_profiles sp ON sp.user_id = o.seller_id
		LEFT JOIN stalls st ON st.id = sp.stall_id
		`+where, args...)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (Order, error) {
		var o Order
		return o, row.Scan(&o.ID, &o.ListingID, &o.ListingTitle, &o.CustomerName, &o.CustomerPhone, &o.SellerName, &o.SellerPhone,
			&o.StallCode, &o.Quantity, &o.UnitPriceTetri, &o.Delivery, &o.DeliveryAddress, &o.DeliveryFeeTetri, &o.TotalTetri,
			&o.Status, &o.CreatedAt)
	})
}

func (s *Server) createOrder(w http.ResponseWriter, r *http.Request, u *User) error {
	var in struct {
		ListingID       int64  `json:"listingId"`
		Quantity        int    `json:"quantity"`
		Delivery        bool   `json:"delivery"`
		DeliveryAddress string `json:"deliveryAddress"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	if in.Quantity < 1 || in.Quantity > 1000 {
		return badRequest("Invalid quantity")
	}
	address, err := trimmed(in.DeliveryAddress, "Delivery address", 0, 300)
	if err != nil {
		return err
	}
	if in.Delivery && len([]rune(address)) < 5 {
		return badRequest("Enter your delivery address")
	}
	var fee int64
	if in.Delivery {
		fee = s.cfg.DeliveryFee
	}
	var orderID int64
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var sellerID, price int64
		var stock int
		var active, sellerActive bool
		err := tx.QueryRow(r.Context(), `
			SELECT l.seller_id, l.price_tetri, l.stock, l.active, COALESCE(u.registration_paid_until > now(), false)
			FROM listings l JOIN users u ON u.id = l.seller_id WHERE l.id = $1 FOR UPDATE OF l`, in.ListingID).
			Scan(&sellerID, &price, &stock, &active, &sellerActive)
		if err != nil {
			return err
		}
		if !active || !sellerActive {
			return errStatus(http.StatusConflict, "This item is no longer available")
		}
		if sellerID == u.ID {
			return badRequest("You can't order your own item")
		}
		if stock < in.Quantity {
			return errStatus(http.StatusConflict, "Only %d left in stock", stock)
		}
		if _, err := tx.Exec(r.Context(), `UPDATE listings SET stock = stock - $2 WHERE id = $1`, in.ListingID, in.Quantity); err != nil {
			return err
		}
		return tx.QueryRow(r.Context(), `
			INSERT INTO orders (customer_id, seller_id, listing_id, quantity, unit_price_tetri, delivery, delivery_address,
			                    delivery_fee_tetri, total_tetri)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id`,
			u.ID, sellerID, in.ListingID, in.Quantity, price, in.Delivery, address, fee, price*int64(in.Quantity)+fee).Scan(&orderID)
	})
	if err != nil {
		return err
	}
	orders, err := s.queryOrders(r.Context(), "WHERE o.id = $1", orderID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusCreated, orders[0])
	return nil
}

func (s *Server) myOrders(w http.ResponseWriter, r *http.Request, u *User) error {
	orders, err := s.queryOrders(r.Context(), "WHERE o.customer_id = $1 ORDER BY o.created_at DESC LIMIT 100", u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, orders)
	return nil
}

func (s *Server) incomingOrders(w http.ResponseWriter, r *http.Request, u *User) error {
	orders, err := s.queryOrders(r.Context(), "WHERE o.seller_id = $1 ORDER BY o.created_at DESC LIMIT 200", u.ID)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, orders)
	return nil
}

var sellerTransitions = map[string][]string{
	"pending":          {"confirmed", "cancelled"},
	"confirmed":        {"out_for_delivery", "completed", "cancelled"},
	"out_for_delivery": {"completed", "cancelled"},
}

func (s *Server) updateOrder(w http.ResponseWriter, r *http.Request, u *User) error {
	id, err := pathID(r)
	if err != nil {
		return err
	}
	var in struct {
		Status string `json:"status"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return err
	}
	err = pgx.BeginFunc(r.Context(), s.db, func(tx pgx.Tx) error {
		var customerID, sellerID, listingID int64
		var status string
		var qty int
		if err := tx.QueryRow(r.Context(), `SELECT customer_id, seller_id, listing_id, status, quantity FROM orders WHERE id = $1 FOR UPDATE`, id).
			Scan(&customerID, &sellerID, &listingID, &status, &qty); err != nil {
			return err
		}
		switch {
		case u.ID == sellerID || u.Role == "admin":
			if !slices.Contains(sellerTransitions[status], in.Status) {
				return badRequest("Can't change order from %s to %s", status, in.Status)
			}
		case u.ID == customerID:
			if status != "pending" || in.Status != "cancelled" {
				return badRequest("You can only cancel an order before the seller confirms it")
			}
		default:
			return pgx.ErrNoRows
		}
		if _, err := tx.Exec(r.Context(), `UPDATE orders SET status = $2 WHERE id = $1`, id, in.Status); err != nil {
			return err
		}
		if in.Status == "cancelled" {
			_, err := tx.Exec(r.Context(), `UPDATE listings SET stock = stock + $2 WHERE id = $1`, listingID, qty)
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	orders, err := s.queryOrders(r.Context(), "WHERE o.id = $1", id)
	if err != nil {
		return err
	}
	writeJSON(w, http.StatusOK, orders[0])
	return nil
}
