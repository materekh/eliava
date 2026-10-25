package server

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"net/http"
	"testing"
	"time"
)

func jpegWithOrientation(t *testing.T, w, h int, orientation uint16) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := 0; x < w; x++ {
		for y := 0; y < h; y++ {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 100, 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	tiff := []byte("II*\x00\x08\x00\x00\x00\x01\x00")
	entry := make([]byte, 12)
	binary.LittleEndian.PutUint16(entry[0:], 0x0112)
	binary.LittleEndian.PutUint16(entry[2:], 3)
	binary.LittleEndian.PutUint32(entry[4:], 1)
	binary.LittleEndian.PutUint16(entry[8:], orientation)
	tiff = append(append(tiff, entry...), 0, 0, 0, 0)
	payload := append([]byte("Exif\x00\x00"), tiff...)
	seg := []byte{0xFF, 0xE1, 0, 0}
	binary.BigEndian.PutUint16(seg[2:], uint16(len(payload)+2))
	seg = append(seg, payload...)
	data := buf.Bytes()
	return append(append(append([]byte{}, data[:2]...), seg...), data[2:]...)
}

func TestCleanImageStripsExifAndRotates(t *testing.T) {
	in := jpegWithOrientation(t, 40, 20, 6)
	if exifOrientation(in) != 6 {
		t.Fatalf("test image orientation = %d", exifOrientation(in))
	}
	out, mime, _, err := cleanImage(in, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	if mime != "image/jpeg" || bytes.Contains(out, []byte("Exif")) {
		t.Fatalf("metadata not stripped (mime %s)", mime)
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Width != 20 || cfg.Height != 40 {
		t.Fatalf("rotation not applied: %dx%d", cfg.Width, cfg.Height)
	}
}

func TestCleanImageShrinksLargePhotos(t *testing.T) {
	out, _, _, err := cleanImage(jpegWithOrientation(t, 3200, 1000, 1), "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	cfg, _, _ := image.DecodeConfig(bytes.NewReader(out))
	if cfg.Width != maxImageSide || cfg.Height != 500 {
		t.Fatalf("got %dx%d", cfg.Width, cfg.Height)
	}
}

func TestRateLimiter(t *testing.T) {
	l := newRateLimiter()
	for i := 0; i < 3; i++ {
		if !l.allow("k", 3, time.Minute) {
			t.Fatalf("attempt %d blocked", i+1)
		}
	}
	if l.allow("k", 3, time.Minute) {
		t.Fatal("4th attempt allowed")
	}
	if !l.allow("other", 3, time.Minute) {
		t.Fatal("other key blocked")
	}
}

func TestAdminEmailCannotBeClaimedByRegistering(t *testing.T) {
	ts := newTestServer(t)
	c := &client{t: t, base: ts.URL}
	c.do("POST", "/api/auth/register", map[string]any{
		"role": "customer", "name": "Attacker", "email": "admin@example.com", "phone": "+995 555 000 111",
		"password": "attacker123", "acceptTerms": true,
	}, http.StatusConflict, nil)
	c.do("POST", "/api/auth/login", map[string]any{"email": "admin@example.com", "password": "attacker123"}, http.StatusUnauthorized, nil)
	// A normal sign-up never becomes admin.
	u := register(t, ts.URL, "customer", "someone@example.com")
	u.do("GET", "/api/admin/reveals", nil, http.StatusForbidden, nil)
}

func TestDeleteAccount(t *testing.T) {
	ts := newTestServer(t)
	u := register(t, ts.URL, "customer", "leaver@example.com")
	u.do("DELETE", "/api/me", map[string]any{"password": "wrong-pass"}, http.StatusForbidden, nil)
	u.do("DELETE", "/api/me", map[string]any{"password": "secret123"}, http.StatusNoContent, nil)
	u.do("GET", "/api/me", nil, http.StatusUnauthorized, nil)
	c := &client{t: t, base: ts.URL}
	c.do("POST", "/api/auth/login", map[string]any{"email": "leaver@example.com", "password": "secret123"}, http.StatusUnauthorized, nil)
}

func TestNormalizePhone(t *testing.T) {
	ok := map[string]string{
		"+995 555 123 456":   "+995 555 12 34 56",
		"555123456":          "+995 555 12 34 56",
		"00995 599-11-22-33": "+995 599 11 22 33",
		"(032) 2 11 22 33":   "+995 322 11 22 33",
	}
	for in, want := range ok {
		if got, err := normalizePhone(in); err != nil || got != want {
			t.Errorf("normalizePhone(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, in := range []string{"12345", "+1 202 555 0143", "655123456", "555 12a 456"} {
		if _, err := normalizePhone(in); err == nil {
			t.Errorf("normalizePhone(%q) accepted", in)
		}
	}
}

func TestReports(t *testing.T) {
	ts := newTestServer(t)
	customer := register(t, ts.URL, "customer", "reporter@example.com")
	tech := register(t, ts.URL, "technician", "reported@example.com")
	var me User
	tech.do("GET", "/api/me", nil, http.StatusOK, &me)
	tech.do("POST", "/api/reports", map[string]any{"targetType": "user", "targetId": me.ID, "reason": "fake"}, http.StatusBadRequest, nil)
	customer.do("POST", "/api/reports", map[string]any{"targetType": "user", "targetId": me.ID, "reason": "nope"}, http.StatusBadRequest, nil)
	customer.do("POST", "/api/reports", map[string]any{"targetType": "listing", "targetId": 999999, "reason": "scam"}, http.StatusNotFound, nil)
	customer.do("POST", "/api/reports", map[string]any{"targetType": "user", "targetId": me.ID, "reason": "unsafe", "details": "Did not show up, asked for money upfront"}, http.StatusCreated, nil)
	customer.do("GET", "/api/admin/reports", nil, http.StatusForbidden, nil)
	admin := login(t, ts.URL, "admin@example.com", "admin-secret-123")
	var list []Report
	admin.do("GET", "/api/admin/reports", nil, http.StatusOK, &list)
	if len(list) != 1 || list[0].Reason != "unsafe" || list[0].ReportCount != 1 || list[0].TargetType != "user" {
		t.Fatalf("reports: %+v", list)
	}
	admin.do("POST", fmt.Sprintf("/api/admin/reports/%d/resolve", list[0].ID), nil, http.StatusOK, nil)
	admin.do("GET", "/api/admin/reports", nil, http.StatusOK, &list)
	if len(list) != 0 {
		t.Fatalf("resolved report still listed: %+v", list)
	}
}
