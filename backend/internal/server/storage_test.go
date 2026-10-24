package server

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSupabaseStorage(t *testing.T) {
	var got []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		if r.Header.Get("Authorization") != "Bearer key" || r.Header.Get("apikey") != "key" {
			t.Errorf("missing auth headers on %s %s", r.Method, r.URL.Path)
		}
		got = append(got, r.Method+" "+r.URL.Path+" "+r.Header.Get("Content-Type")+" "+string(body))
		if r.URL.Path == "/storage/v1/bucket" {
			w.WriteHeader(http.StatusBadRequest)
			io.WriteString(w, `{"statusCode":"409","error":"Duplicate","message":"The resource already exists"}`)
			return
		}
		io.WriteString(w, `{}`)
	}))
	defer ts.Close()
	st := SupabaseStorage{URL: ts.URL + "/", ServiceKey: "key", Bucket: "media"}
	s := &Server{cfg: Config{Storage: st}}
	ctx := context.Background()

	if err := st.EnsureBucket(ctx); err != nil {
		t.Fatalf("existing bucket should be fine: %v", err)
	}
	n, err := s.storeFile(ctx, "posts/a.jpg", "image/jpeg", strings.NewReader("jpegdata"), 8, 100)
	if err != nil || n != 8 {
		t.Fatalf("storeFile = %d, %v", n, err)
	}
	s.removeFiles([]savedFile{{rel: "posts/a.jpg"}})
	if u := s.mediaURL("posts/a.jpg"); u != ts.URL+"/storage/v1/object/public/media/posts/a.jpg" {
		t.Fatalf("mediaURL = %s", u)
	}
	want := []string{
		`POST /storage/v1/bucket application/json {"id":"media","name":"media","public":true}`,
		`POST /storage/v1/object/media/posts/a.jpg image/jpeg jpegdata`,
		`DELETE /storage/v1/object/media application/json {"prefixes":["posts/a.jpg"]}`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("requests:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}

	if (&Server{}).mediaURL("posts/a.jpg") != "/uploads/posts/a.jpg" {
		t.Fatal("local mediaURL changed")
	}
}
