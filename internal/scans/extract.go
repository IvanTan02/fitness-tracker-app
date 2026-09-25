package scans

import (
	"encoding/json"
	"io"
	"log"
	"net/http"

	"github.com/ivantan02/fitness-tracker-app/internal/auth"
	"github.com/ivantan02/fitness-tracker-app/internal/platform"
)

// maxUploadBytes caps accepted images well under the free-tier request
// budget; a huge upload should never be allowed to eat the request budget.
const maxUploadBytes = 10 << 20 // 10MB

var allowedImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
	"image/heic": true,
}

// extractHandler accepts a multipart image upload, sends it to Gemini for
// vision extraction, and returns the parsed measurements as JSON. Extraction
// failures return an error the frontend can show — they must never block
// manual entry, so this handler never panics or hangs indefinitely.
func extractHandler(gemini *platform.GeminiClient, limiter *extractRateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)

		if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
			http.Error(w, "request too large or malformed", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("photo")
		if err != nil {
			http.Error(w, "missing 'photo' file field", http.StatusBadRequest)
			return
		}
		defer file.Close()

		mimeType := header.Header.Get("Content-Type")
		if !allowedImageTypes[mimeType] {
			http.Error(w, "unsupported image type; use JPEG, PNG, WEBP, or HEIC", http.StatusBadRequest)
			return
		}

		imageBytes, err := io.ReadAll(file)
		if err != nil {
			http.Error(w, "failed to read uploaded file", http.StatusBadRequest)
			return
		}

		extracted, err := gemini.ExtractScan(r.Context(), imageBytes, mimeType)
		if err != nil {
			// Degrade gracefully: log the detail server-side, but the
			// frontend only needs to know to fall back to manual entry.
			log.Printf("extraction failed: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadGateway)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "couldn't read the photo automatically — please enter the numbers manually",
			})
			return
		}

		limiter.record(auth.UserID(r.Context()))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(extracted)
	}
}
