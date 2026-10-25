package server

import (
	"bytes"
	"encoding/binary"
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
