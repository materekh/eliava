package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SupabaseStorage stores uploads in a public Supabase Storage bucket instead of the local disk.
type SupabaseStorage struct {
	URL        string
	ServiceKey string
	Bucket     string
}

func (c SupabaseStorage) Enabled() bool { return c.URL != "" && c.ServiceKey != "" }

var storageHTTP = &http.Client{Timeout: 5 * time.Minute}

func (c SupabaseStorage) base() string { return strings.TrimRight(c.URL, "/") + "/storage/v1" }

func (c SupabaseStorage) do(ctx context.Context, method, endpoint, contentType string, body io.Reader, size int64) (int, []byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.base()+endpoint, body)
	if err != nil {
		return 0, nil, err
	}
	if size >= 0 {
		req.ContentLength = size
	}
	req.Header.Set("Authorization", "Bearer "+c.ServiceKey)
	req.Header.Set("apikey", c.ServiceKey)
	req.Header.Set("Content-Type", contentType)
	res, err := storageHTTP.Do(req)
	if err != nil {
		return 0, nil, err
	}
	defer res.Body.Close()
	msg, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
	return res.StatusCode, msg, nil
}

func (c SupabaseStorage) objectPath(key string) string {
	return "/object/" + url.PathEscape(c.Bucket) + "/" + key
}

// EnsureBucket creates the public bucket if it doesn't exist yet.
func (c SupabaseStorage) EnsureBucket(ctx context.Context) error {
	body, _ := json.Marshal(map[string]any{"id": c.Bucket, "name": c.Bucket, "public": true})
	status, msg, err := c.do(ctx, http.MethodPost, "/bucket", "application/json", bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}
	if status < 300 || strings.Contains(strings.ToLower(string(msg)), "already exists") {
		return nil
	}
	return fmt.Errorf("create bucket %q: %d %s", c.Bucket, status, msg)
}

func (c SupabaseStorage) put(ctx context.Context, key, mime string, body io.Reader, size int64) error {
	status, msg, err := c.do(ctx, http.MethodPost, c.objectPath(key), mime, body, size)
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("upload %s: %d %s", key, status, msg)
	}
	return nil
}

func (c SupabaseStorage) remove(ctx context.Context, keys []string) error {
	body, _ := json.Marshal(map[string][]string{"prefixes": keys})
	status, msg, err := c.do(ctx, http.MethodDelete, "/object/"+url.PathEscape(c.Bucket), "application/json", bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return err
	}
	if status >= 300 {
		return fmt.Errorf("delete: %d %s", status, msg)
	}
	return nil
}

func (c SupabaseStorage) publicURL(key string) string {
	return c.base() + "/object/public/" + url.PathEscape(c.Bucket) + "/" + key
}

// storeFile saves an already-validated upload under key ("posts/abc.jpg") and returns its size.
func (s *Server) storeFile(ctx context.Context, key, mime string, src io.Reader, size, limit int64) (int64, error) {
	if s.cfg.Storage.Enabled() {
		return size, s.cfg.Storage.put(ctx, key, mime, io.LimitReader(src, size), size)
	}
	full := filepath.Join(s.cfg.UploadDir, filepath.FromSlash(key))
	if err := ensureDir(filepath.Dir(full)); err != nil {
		return 0, err
	}
	dst, err := os.Create(full)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err == nil && n > limit {
		err = badRequest("File is too large")
	}
	if err != nil {
		os.Remove(full)
		return 0, err
	}
	return n, nil
}

func (s *Server) removeFiles(files []savedFile) {
	if len(files) == 0 {
		return
	}
	if s.cfg.Storage.Enabled() {
		keys := make([]string, len(files))
		for i, f := range files {
			keys[i] = f.rel
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = s.cfg.Storage.remove(ctx, keys)
		return
	}
	for _, f := range files {
		os.Remove(filepath.Join(s.cfg.UploadDir, filepath.FromSlash(f.rel)))
	}
}

func (s *Server) mediaURL(rel string) string {
	if s.cfg.Storage.Enabled() {
		return s.cfg.Storage.publicURL(rel)
	}
	return "/uploads/" + rel
}
