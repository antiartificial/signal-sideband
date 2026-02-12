package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"signal-sideband/pkg/ai"
	"signal-sideband/pkg/digest"
	"signal-sideband/pkg/store"
)

type Server struct {
	httpServer *http.Server
	handlers   *Handlers
}

func NewServer(s *store.Store, embedder ai.Embedder, generator *digest.Generator, port string, mediaPath string, webDir ...string) *Server {
	h := NewHandlers(s, embedder, generator, mediaPath)

	mux := http.NewServeMux()

	// Messages
	mux.HandleFunc("GET /api/messages", h.GetMessages)
	mux.HandleFunc("GET /api/messages/search", h.SearchMessages)

	// Groups
	mux.HandleFunc("GET /api/groups", h.GetGroups)

	// Digests
	mux.HandleFunc("GET /api/digests", h.GetDigests)
	mux.HandleFunc("GET /api/digests/{id}", h.GetDigest)
	mux.HandleFunc("POST /api/digests/generate", h.GenerateDigest)

	// URLs
	mux.HandleFunc("GET /api/urls", h.GetURLs)

	// Media
	mux.HandleFunc("GET /api/media", h.GetMedia)
	mux.HandleFunc("GET /api/media/{id}", h.ServeMedia)

	// Stats
	mux.HandleFunc("GET /api/stats", h.GetStats)

	// Health
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Serve static frontend if web directory exists
	if len(webDir) > 0 && webDir[0] != "" {
		dir := webDir[0]
		fs := http.FileServer(http.Dir(dir))
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			// Try the actual file first; fall back to index.html for SPA routing
			if _, err := os.Stat(dir + r.URL.Path); os.IsNotExist(err) {
				http.ServeFile(w, r, dir+"/index.html")
				return
			}
			fs.ServeHTTP(w, r)
		})
	}

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      corsMiddleware(mux),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 120 * time.Second,
		},
		handlers: h,
	}
}

func (s *Server) Start() error {
	log.Printf("API server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
