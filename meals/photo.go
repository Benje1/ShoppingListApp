package meals

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"

	sqlc "weekly-shopping-app/database/sqlc"
	"weekly-shopping-app/internal/api/httpx"
	"weekly-shopping-app/internal/logger"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Meal photo upload/serving.
//
// Uploaded photos are stored content-addressed (SHA-256 of the bytes) in the
// meal_photos table, so they survive redeploys on hosts with an ephemeral
// filesystem (e.g. Render) without needing separate object storage or secrets.
// The upload endpoint only hosts the file and returns its URL; the web app then
// stores that URL in the meal's photo_url via the normal create/update calls.
//
// Swapping to a CDN/object store later only changes these two handlers — the
// API surface (POST /meals/photo/upload → {"url": ...}) stays the same.

const (
	// maxPhotoBytes caps an upload. The client downscales to ~1MB; 5MB is a safe
	// ceiling that still rejects accidental full-resolution uploads.
	maxPhotoBytes = 5 << 20
	// photoFormField is the multipart file part name the client sends.
	photoFormField = "photo"
)

// allowedPhotoTypes is the set of image content types we accept and serve back.
var allowedPhotoTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// handleMealPhotoUpload reads a multipart/form-data upload with a single file
// part named "photo", validates and stores it, and returns {"url": <hosted>}.
func handleMealPhotoUpload(db *pgxpool.Pool) httpx.AppHandler {
	return func(w http.ResponseWriter, r *http.Request) (any, error) {
		// Cap the request body defensively before touching the multipart parser.
		r.Body = http.MaxBytesReader(w, r.Body, maxPhotoBytes+(1<<16))
		if err := r.ParseMultipartForm(maxPhotoBytes); err != nil {
			return nil, httpx.NewClientError(fmt.Errorf("could not read upload (max %d MB): %w", maxPhotoBytes>>20, err))
		}
		defer func() {
			if r.MultipartForm != nil {
				_ = r.MultipartForm.RemoveAll()
			}
		}()

		file, header, err := r.FormFile(photoFormField)
		if err != nil {
			return nil, httpx.NewClientError(fmt.Errorf("a %q file part is required", photoFormField))
		}
		defer file.Close()

		data, err := io.ReadAll(io.LimitReader(file, maxPhotoBytes+1))
		if err != nil {
			return nil, logger.WithStack(err)
		}
		if len(data) == 0 {
			return nil, httpx.NewClientError(errors.New("uploaded file is empty"))
		}
		if len(data) > maxPhotoBytes {
			return nil, httpx.NewClientError(fmt.Errorf("file exceeds the %d MB limit", maxPhotoBytes>>20))
		}

		contentType, ok := sniffImageType(data, header.Header.Get("Content-Type"))
		if !ok {
			return nil, httpx.NewClientError(fmt.Errorf("unsupported image type (allowed: jpeg, png, webp)"))
		}

		sum := sha256.Sum256(data)
		id := hex.EncodeToString(sum[:])

		if err := sqlc.New(db).UpsertMealPhoto(r.Context(), sqlc.UpsertMealPhotoParams{
			ID:          id,
			ContentType: contentType,
			Bytes:       data,
			ByteSize:    int32(len(data)),
		}); err != nil {
			return nil, logger.WithStack(err)
		}

		return map[string]string{"url": photoURL(r, id)}, nil
	}
}

// handleMealPhotoServe streams a stored photo back with a long, immutable cache
// header. It is public: the URL is content-addressed by an unguessable SHA-256,
// which acts as the access capability (like an object-store signed URL), so an
// <img src> can load it without carrying the session cookie cross-origin.
func handleMealPhotoServe(db *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("hash")))
		if !isHexSHA256(id) {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}

		// A content-addressed image never changes, so honour conditional requests.
		etag := `"` + id + `"`
		if match := r.Header.Get("If-None-Match"); match != "" && strings.Contains(match, id) {
			w.Header().Set("ETag", etag)
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.WriteHeader(http.StatusNotModified)
			return
		}

		photo, err := sqlc.New(db).GetMealPhoto(r.Context(), id)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				http.Error(w, "not found", http.StatusNotFound)
				return
			}
			logger.Error("serve meal photo", "err", err)
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", photo.ContentType)
		w.Header().Set("Content-Length", strconv.Itoa(len(photo.Bytes)))
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		w.Header().Set("ETag", etag)
		if _, err := w.Write(photo.Bytes); err != nil {
			logger.Warn("meal photo write failed", "err", err)
		}
	}
}

// sniffImageType determines the image type from the bytes themselves (not the
// client-declared type) via magic numbers, falling back to the multipart part's
// declared Content-Type only when it is one we allow. Returns ok=false for
// anything outside the allowed set.
func sniffImageType(data []byte, declared string) (string, bool) {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg", true
	case len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n":
		return "image/png", true
	case len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP":
		return "image/webp", true
	}
	declared = strings.TrimSpace(strings.ToLower(declared))
	if allowedPhotoTypes[declared] {
		return declared, true
	}
	return "", false
}

// isHexSHA256 reports whether s is a 64-char lowercase hex string (a SHA-256).
func isHexSHA256(s string) bool {
	if len(s) != 64 {
		return false
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

// photoURL builds the absolute hosted URL for a stored photo. It prefers the
// PUBLIC_API_BASE_URL env var (set this to the API's public origin behind a
// proxy/CDN); otherwise it derives the origin from the incoming request.
func photoURL(r *http.Request, id string) string {
	base := strings.TrimRight(os.Getenv("PUBLIC_API_BASE_URL"), "/")
	if base == "" {
		base = requestBaseURL(r)
	}
	return base + "/meals/photo/get?hash=" + id
}

// requestBaseURL reconstructs the request's own scheme+host, honouring the
// proxy headers Render (and most PaaS proxies) set in front of the app.
func requestBaseURL(r *http.Request) string {
	scheme := "http"
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		scheme = strings.TrimSpace(strings.Split(proto, ",")[0])
	} else if r.TLS != nil {
		scheme = "https"
	}
	host := r.Host
	if fwd := r.Header.Get("X-Forwarded-Host"); fwd != "" {
		host = strings.TrimSpace(strings.Split(fwd, ",")[0])
	}
	return scheme + "://" + host
}
