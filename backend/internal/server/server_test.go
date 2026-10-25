package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Integration tests need a PostGIS database: TEST_DATABASE_URL=postgres://postgres:devpass@localhost:5432/eliava_test
func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public`); err != nil {
		t.Fatal(err)
	}
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	if err := BootstrapAdmin(ctx, pool, "admin@example.com", "admin-secret-123"); err != nil {
		t.Fatal(err)
	}
	srv := New(pool, Config{
		UploadDir: t.TempDir(), DisableRateLimit: true, PaymentsMode: "mock", TermsVersion: "test",
		RegistrationFee: 2000, RegistrationDays: 30, RevealPrice: 200, DeliveryFee: 500, BoostDays: 30,
	})
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)
	return ts
}

type client struct {
	t     *testing.T
	base  string
	token string
}

func (c *client) do(method, path string, body any, wantStatus int, out any) {
	c.t.Helper()
	var buf bytes.Buffer
	contentType := "application/json"
	switch b := body.(type) {
	case nil:
	case *multipartBody:
		buf, contentType = b.buf, b.contentType
	default:
		json.NewEncoder(&buf).Encode(b)
	}
	req, _ := http.NewRequest(method, c.base+path, &buf)
	req.Header.Set("Content-Type", contentType)
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer resp.Body.Close()
	var raw json.RawMessage
	json.NewDecoder(resp.Body).Decode(&raw)
	if resp.StatusCode != wantStatus {
		c.t.Fatalf("%s %s: got %d, want %d: %s", method, path, resp.StatusCode, wantStatus, raw)
	}
	if out != nil {
		if err := json.Unmarshal(raw, out); err != nil {
			c.t.Fatalf("%s %s: decode: %v (%s)", method, path, err, raw)
		}
	}
}

type multipartBody struct {
	buf         bytes.Buffer
	contentType string
}

func form(fields map[string]string, images int) *multipartBody {
	mb := &multipartBody{}
	w := multipart.NewWriter(&mb.buf)
	for k, v := range fields {
		w.WriteField(k, v)
	}
	for i := 0; i < images; i++ {
		img := image.NewRGBA(image.Rect(0, 0, 4, 4))
		img.Set(0, 0, color.RGBA{255, 0, 0, 255})
		fw, _ := w.CreateFormFile("media", fmt.Sprintf("photo%d.png", i))
		png.Encode(fw, img)
	}
	w.Close()
	mb.contentType = w.FormDataContentType()
	return mb
}

func register(t *testing.T, base, role, email string) *client {
	c := &client{t: t, base: base}
	var out struct {
		Token string `json:"token"`
	}
	c.do("POST", "/api/auth/register", map[string]any{
		"role": role, "name": role + " " + email, "email": email, "phone": "+995 555 123 456",
		"password": "secret123", "acceptTerms": true,
	}, http.StatusCreated, &out)
	c.token = out.Token
	return c
}

func login(t *testing.T, base, email, password string) *client {
	c := &client{t: t, base: base}
	var out struct {
		Token string `json:"token"`
	}
	c.do("POST", "/api/auth/login", map[string]any{"email": email, "password": password}, http.StatusOK, &out)
	c.token = out.Token
	return c
}

func (c *client) payRegistration() {
	c.do("POST", "/api/wallet/topup", map[string]any{"amountTetri": 10000}, http.StatusOK, nil)
	c.do("POST", "/api/registration/pay", nil, http.StatusOK, nil)
}

func categoryID(t *testing.T, c *client, slug string) int64 {
	var cats []Category
	c.do("GET", "/api/categories", nil, http.StatusOK, &cats)
	for _, cat := range cats {
		if cat.Slug == slug {
			return cat.ID
		}
	}
	t.Fatalf("category %s not found", slug)
	return 0
}

func TestRepairFlow(t *testing.T) {
	ts := newTestServer(t)
	customer := register(t, ts.URL, "customer", "cust@example.com")
	tech := register(t, ts.URL, "technician", "tech@example.com")
	far := register(t, ts.URL, "technician", "far@example.com")
	admin := login(t, ts.URL, "admin@example.com", "admin-secret-123")
	ac := categoryID(t, customer, "air-conditioner")

	// Unpaid technicians are hidden from search.
	tech.do("PUT", "/api/me/technician-profile", map[string]any{
		"bio": "AC expert", "city": "Tbilisi", "location": Point{41.72, 44.79}, "serviceRadiusKm": 15, "categoryIds": []int64{ac},
	}, http.StatusOK, nil)
	var techs []Technician
	customer.do("GET", "/api/technicians?lat=41.71&lng=44.80&radius_km=10", nil, http.StatusOK, &techs)
	if len(techs) != 0 {
		t.Fatalf("unpaid technician should be hidden, got %d", len(techs))
	}
	tech.payRegistration()
	far.do("PUT", "/api/me/technician-profile", map[string]any{
		"location": Point{41.64, 41.63}, "serviceRadiusKm": 15, "categoryIds": []int64{ac},
	}, http.StatusOK, nil)
	far.payRegistration()

	customer.do("GET", "/api/technicians?lat=41.71&lng=44.80&radius_km=10&category=air-conditioner", nil, http.StatusOK, &techs)
	if len(techs) != 1 || techs[0].Name != "technician tech@example.com" || techs[0].DistanceKM == nil {
		t.Fatalf("nearby search: %+v", techs)
	}
	customer.do("GET", "/api/technicians?lat=41.71&lng=44.80&category=plumbing", nil, http.StatusOK, &techs)
	if len(techs) != 0 {
		t.Fatalf("category filter: %+v", techs)
	}

	// Posts need a photo and a real description.
	fields := map[string]string{
		"categoryId": fmt.Sprint(ac), "title": "AC not cooling", "lat": "41.715", "lng": "44.80", "hasParts": "false",
		"description": "My Samsung air conditioner blows warm air and makes a clicking noise.",
	}
	customer.do("POST", "/api/posts", form(fields, 0), http.StatusBadRequest, nil)
	var post Post
	customer.do("POST", "/api/posts", form(fields, 2), http.StatusCreated, &post)
	if len(post.Media) != 2 || post.Flagged || post.Estimate.MinTetri <= 0 || post.Estimate.MaxTetri < post.Estimate.MinTetri {
		t.Fatalf("post: %+v", post)
	}
	var withParts Estimate
	customer.do("POST", "/api/estimate", map[string]any{"categoryId": ac, "hasParts": true, "location": Point{41.715, 44.80}}, http.StatusOK, &withParts)
	if withParts.MaxTetri >= post.Estimate.MaxTetri {
		t.Fatalf("having parts should lower the estimate: %+v vs %+v", withParts, post.Estimate)
	}

	// Posting the same problem again gets flagged and hidden from technicians.
	var dup Post
	fields["title"] = "AC not cooling!!"
	customer.do("POST", "/api/posts", form(fields, 1), http.StatusCreated, &dup)
	if !dup.Flagged || dup.DuplicateOf == nil || *dup.DuplicateOf != post.ID {
		t.Fatalf("duplicate not flagged: %+v", dup)
	}

	var nearby []Post
	tech.do("GET", "/api/posts/nearby", nil, http.StatusOK, &nearby)
	if len(nearby) != 1 || nearby[0].ID != post.ID || nearby[0].Contact != nil || nearby[0].Address != "" ||
		nearby[0].Location.Lat*100 != float64(int64(nearby[0].Location.Lat*100)) {
		t.Fatalf("nearby posts: %+v", nearby)
	}
	far.do("GET", "/api/posts/nearby", nil, http.StatusOK, &nearby)
	if len(nearby) != 0 {
		t.Fatalf("far technician should not see the post: %+v", nearby)
	}

	// Revealing the number charges once and is logged.
	var revealed Post
	tech.do("POST", fmt.Sprintf("/api/posts/%d/reveal", post.ID), nil, http.StatusOK, &revealed)
	tech.do("POST", fmt.Sprintf("/api/posts/%d/reveal", post.ID), nil, http.StatusOK, &revealed)
	if revealed.Contact == nil || revealed.Contact.Phone == "" {
		t.Fatalf("contact not revealed: %+v", revealed)
	}
	var me User
	tech.do("GET", "/api/me", nil, http.StatusOK, &me)
	if me.BalanceTetri != 10000-2000-200 {
		t.Fatalf("balance after reveal = %d", me.BalanceTetri)
	}
	var viewers []Viewer
	customer.do("GET", fmt.Sprintf("/api/posts/%d/viewers", post.ID), nil, http.StatusOK, &viewers)
	if len(viewers) != 1 || viewers[0].Name != "technician tech@example.com" {
		t.Fatalf("viewers: %+v", viewers)
	}
	var logs []RevealLog
	customer.do("GET", "/api/admin/reveals", nil, http.StatusForbidden, nil)
	admin.do("GET", "/api/admin/reveals", nil, http.StatusOK, &logs)
	if len(logs) != 1 || logs[0].TechnicianEmail != "tech@example.com" || logs[0].PostID != post.ID {
		t.Fatalf("admin reveal log: %+v", logs)
	}
	var flagged []Post
	admin.do("GET", "/api/admin/flagged-posts", nil, http.StatusOK, &flagged)
	if len(flagged) != 1 || flagged[0].ID != dup.ID {
		t.Fatalf("flagged: %+v", flagged)
	}

	// A broke technician can't reveal.
	broke := register(t, ts.URL, "technician", "broke@example.com")
	broke.do("POST", fmt.Sprintf("/api/posts/%d/reveal", post.ID), nil, http.StatusPaymentRequired, nil)
}

func TestMarketFlow(t *testing.T) {
	ts := newTestServer(t)
	seller := register(t, ts.URL, "seller", "seller@example.com")
	seller2 := register(t, ts.URL, "seller", "seller2@example.com")
	customer := register(t, ts.URL, "customer", "cust@example.com")
	parts := categoryID(t, seller, "fridge-parts")

	var stalls []Stall
	customer.do("GET", "/api/stalls", nil, http.StatusOK, &stalls)
	if len(stalls) == 0 {
		t.Fatal("no stalls seeded")
	}
	seller.do("PUT", "/api/me/seller-profile", map[string]any{"shopName": "Gia Parts", "stallId": stalls[0].ID}, http.StatusOK, nil)
	seller2.do("PUT", "/api/me/seller-profile", map[string]any{"shopName": "Nino Shop", "stallId": stalls[1].ID}, http.StatusOK, nil)

	listingForm := map[string]string{"categoryId": fmt.Sprint(parts), "title": "Fridge compressor", "price": "150.50", "stock": "2"}
	seller.do("POST", "/api/listings", form(listingForm, 1), http.StatusPaymentRequired, nil)
	seller.payRegistration()
	seller2.payRegistration()
	var listing Listing
	seller.do("POST", "/api/listings", form(listingForm, 1), http.StatusCreated, &listing)
	if listing.PriceTetri != 15050 || listing.Stock != 2 || *listing.Seller.StallCode != stalls[0].Code {
		t.Fatalf("listing: %+v", listing)
	}
	listingForm["title"] = "Fridge thermostat"
	seller2.do("POST", "/api/listings", form(listingForm, 0), http.StatusCreated, nil)

	var found []Listing
	customer.do("GET", fmt.Sprintf("/api/listings?stall=%d", stalls[0].ID), nil, http.StatusOK, &found)
	if len(found) != 1 || found[0].ID != listing.ID {
		t.Fatalf("stall filter: %+v", found)
	}
	customer.do("GET", "/api/listings?q=compressor", nil, http.StatusOK, &found)
	if len(found) != 1 {
		t.Fatalf("search: %+v", found)
	}

	// Stock goes down with orders and comes back on cancel.
	var order Order
	customer.do("POST", "/api/orders", map[string]any{"listingId": listing.ID, "quantity": 3, "delivery": false}, http.StatusConflict, nil)
	customer.do("POST", "/api/orders", map[string]any{"listingId": listing.ID, "quantity": 2, "delivery": true}, http.StatusBadRequest, nil)
	customer.do("POST", "/api/orders", map[string]any{"listingId": listing.ID, "quantity": 2, "delivery": true, "deliveryAddress": "Vazha-Pshavela 10, Tbilisi"}, http.StatusCreated, &order)
	if order.TotalTetri != 2*15050+500 || order.Status != "pending" {
		t.Fatalf("order: %+v", order)
	}
	customer.do("GET", fmt.Sprintf("/api/listings/%d", listing.ID), nil, http.StatusOK, &listing)
	if listing.Stock != 0 {
		t.Fatalf("stock after order = %d", listing.Stock)
	}
	customer.do("POST", "/api/orders", map[string]any{"listingId": listing.ID, "quantity": 1}, http.StatusConflict, nil)
	var incoming []Order
	seller.do("GET", "/api/orders/incoming", nil, http.StatusOK, &incoming)
	if len(incoming) != 1 {
		t.Fatalf("incoming: %+v", incoming)
	}
	seller2.do("PATCH", fmt.Sprintf("/api/orders/%d", order.ID), map[string]any{"status": "confirmed"}, http.StatusNotFound, nil)
	seller.do("PATCH", fmt.Sprintf("/api/orders/%d", order.ID), map[string]any{"status": "confirmed"}, http.StatusOK, nil)
	customer.do("PATCH", fmt.Sprintf("/api/orders/%d", order.ID), map[string]any{"status": "cancelled"}, http.StatusBadRequest, nil)
	seller.do("PATCH", fmt.Sprintf("/api/orders/%d", order.ID), map[string]any{"status": "cancelled"}, http.StatusOK, nil)
	customer.do("GET", fmt.Sprintf("/api/listings/%d", listing.ID), nil, http.StatusOK, &listing)
	if listing.Stock != 2 {
		t.Fatalf("stock after cancel = %d", listing.Stock)
	}

	// Paying more than the current top takes first place, and boosted sellers list first.
	var board struct {
		Entries []LeaderboardEntry `json:"entries"`
	}
	seller.do("POST", "/api/boosts", map[string]any{"amountTetri": 500}, http.StatusOK, &board)
	seller2.do("POST", "/api/boosts", map[string]any{"amountTetri": 1000}, http.StatusOK, &board)
	if len(board.Entries) != 2 || board.Entries[0].Name != "seller seller2@example.com" || board.Entries[0].BoostTetri != 1000 {
		t.Fatalf("leaderboard: %+v", board.Entries)
	}
	customer.do("GET", "/api/listings?category=fridge-parts", nil, http.StatusOK, &found)
	if len(found) != 2 || found[0].Seller.ShopName != "Nino Shop" {
		t.Fatalf("boost ordering: %+v", found)
	}
	customer.do("POST", "/api/boosts", map[string]any{"amountTetri": 500}, http.StatusForbidden, nil)
}

func TestRuleEstimate(t *testing.T) {
	tbilisi := ruleEstimate("AC", 8000, 20000, 5000, 40000, false, Point{41.7151, 44.8271})
	region := ruleEstimate("AC", 8000, 20000, 5000, 40000, false, Point{42.0, 43.5})
	owned := ruleEstimate("AC", 8000, 20000, 5000, 40000, true, Point{41.7151, 44.8271})
	if tbilisi.MinTetri != 13000 || tbilisi.MaxTetri != 60000 {
		t.Fatalf("tbilisi: %+v", tbilisi)
	}
	if region.MaxTetri >= tbilisi.MaxTetri || owned.MaxTetri != 20000 {
		t.Fatalf("region %+v owned %+v", region, owned)
	}
}
