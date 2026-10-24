package server

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maxImageBytes   = 10 << 20
	maxVideoBytes   = 100 << 20
	maxUploadFiles  = 10
	maxRequestBytes = 300 << 20
)

type Media struct {
	ID   int64  `json:"id"`
	Kind string `json:"kind"`
	URL  string `json:"url"`
}

type savedFile struct {
	kind, rel, mime string
	size            int64
}

var allowedMime = map[string]struct{ kind, ext string }{
	"image/jpeg":      {"image", ".jpg"},
	"image/png":       {"image", ".png"},
	"image/webp":      {"image", ".webp"},
	"image/gif":       {"image", ".gif"},
	"video/mp4":       {"video", ".mp4"},
	"video/webm":      {"video", ".webm"},
	"video/quicktime": {"video", ".mov"},
}

func parseMultipart(w http.ResponseWriter, r *http.Request) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBytes)
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			return errStatus(http.StatusRequestEntityTooLarge, "Upload is too large")
		}
		return badRequest("Invalid form data")
	}
	return nil
}

// saveUploads stores the files of a parsed multipart field under UploadDir/sub.
// Callers must call removeFiles if they fail to record the files afterwards.
func (s *Server) saveUploads(r *http.Request, field, sub string) ([]savedFile, error) {
	headers := r.MultipartForm.File[field]
	if len(headers) > maxUploadFiles {
		return nil, badRequest("You can upload at most %d files", maxUploadFiles)
	}
	dir := filepath.Join(s.cfg.UploadDir, sub)
	if err := ensureDir(dir); err != nil {
		return nil, err
	}
	var saved []savedFile
	for _, h := range headers {
		f, err := s.saveUpload(h, dir, sub)
		if err != nil {
			s.removeFiles(saved)
			return nil, err
		}
		saved = append(saved, f)
	}
	return saved, nil
}

func (s *Server) saveUpload(h *multipart.FileHeader, dir, sub string) (savedFile, error) {
	src, err := h.Open()
	if err != nil {
		return savedFile{}, err
	}
	defer src.Close()

	head := make([]byte, 512)
	n, err := io.ReadFull(src, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		return savedFile{}, err
	}
	mime := http.DetectContentType(head[:n])
	if strings.HasPrefix(mime, "application/octet-stream") && strings.EqualFold(path.Ext(h.Filename), ".mov") {
		mime = "video/quicktime"
	}
	info, ok := allowedMime[mime]
	if !ok {
		return savedFile{}, badRequest("%s: only JPG, PNG, WEBP, GIF images and MP4, WEBM, MOV videos are allowed", h.Filename)
	}
	limit := int64(maxImageBytes)
	if info.kind == "video" {
		limit = maxVideoBytes
	}
	if h.Size > limit {
		return savedFile{}, badRequest("%s is too large (max %d MB for %ss)", h.Filename, limit>>20, info.kind)
	}
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return savedFile{}, err
	}

	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return savedFile{}, err
	}
	name := hex.EncodeToString(b) + info.ext
	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		return savedFile{}, err
	}
	size, err := io.Copy(dst, io.LimitReader(src, limit+1))
	if cerr := dst.Close(); err == nil {
		err = cerr
	}
	if err == nil && size > limit {
		err = badRequest("%s is too large", h.Filename)
	}
	if err != nil {
		os.Remove(filepath.Join(dir, name))
		return savedFile{}, err
	}
	return savedFile{kind: info.kind, rel: sub + "/" + name, mime: mime, size: size}, nil
}

func (s *Server) removeFiles(files []savedFile) {
	for _, f := range files {
		os.Remove(filepath.Join(s.cfg.UploadDir, filepath.FromSlash(f.rel)))
	}
}

func mediaURL(rel string) string { return "/uploads/" + rel }
