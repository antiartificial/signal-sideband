package api

import (
	"net/http"
	"strings"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func isMediaServingPath(path string) bool {
	// Exempt file-serving endpoints from auth — img/video tags can't send Bearer tokens.
	// These use UUID-based paths that aren't enumerable.
	if path == "/api/potd" {
		return true
	}
	// /api/media/{uuid} and /api/media/{uuid}/thumb
	if strings.HasPrefix(path, "/api/media/") {
		rest := strings.TrimPrefix(path, "/api/media/")
		// Must have a UUID-like segment (not "search" or empty)
		if rest != "" && rest != "search" && !strings.HasPrefix(rest, "search?") {
			return true
		}
	}
	return false
}

func authMiddleware(password string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/health" || strings.HasPrefix(path, "/api/auth/") || r.Method == http.MethodOptions || !strings.HasPrefix(path, "/api/") {
			next.ServeHTTP(w, r)
			return
		}

		// Allow media file-serving without auth (img/video tags can't send headers)
		if r.Method == http.MethodGet && isMediaServingPath(path) {
			next.ServeHTTP(w, r)
			return
		}

		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")
		if !validateToken(token, password) {
			writeError(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		next.ServeHTTP(w, r)
	})
}
