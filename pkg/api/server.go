package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"signal-sideband/pkg/ai"
	"signal-sideband/pkg/cerebro"
	"signal-sideband/pkg/digest"
	"signal-sideband/pkg/media"
	"signal-sideband/pkg/store"
)

type Server struct {
	httpServer *http.Server
	handlers   *Handlers
}

func NewServer(s *store.Store, embedder ai.Embedder, generator *digest.Generator, insightsGen *digest.InsightsGenerator, picGen *media.PicOfDayGenerator, cerebroExtractor *cerebro.Extractor, cerebroEnricher *cerebro.Enricher, port string, authPassword string, mediaPath string, webDir ...string) *Server {
	h := NewHandlers(s, embedder, generator, insightsGen, picGen, cerebroExtractor, cerebroEnricher, mediaPath, authPassword)

	mux := http.NewServeMux()

	// Auth
	mux.HandleFunc("GET /api/auth/status", h.AuthStatus)
	mux.HandleFunc("POST /api/auth/login", h.Login)

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
	mux.HandleFunc("GET /api/media/search", h.SearchMedia)
	mux.HandleFunc("GET /api/media/{id}", h.ServeMedia)
	mux.HandleFunc("GET /api/media/{id}/thumb", h.ServeMediaThumb)

	// Insights
	mux.HandleFunc("POST /api/insights/generate", h.GenerateInsight)

	// Picture of the Day
	mux.HandleFunc("GET /api/potd", h.ServePicOfDay)
	mux.HandleFunc("POST /api/potd/generate", h.GeneratePicOfDay)

	// Cerebro
	mux.HandleFunc("GET /api/cerebro/graph", h.GetCerebroGraph)
	mux.HandleFunc("GET /api/cerebro/concepts/{id}", h.GetCerebroConcept)
	mux.HandleFunc("POST /api/cerebro/concepts/{id}/enrich", h.EnrichCerebroConcept)
	mux.HandleFunc("POST /api/cerebro/extract", h.ExtractCerebro)

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

	var handler http.Handler = mux
	if authPassword != "" {
		handler = authMiddleware(authPassword, handler)
	}
	handler = corsMiddleware(handler)

	return &Server{
		httpServer: &http.Server{
			Addr:         fmt.Sprintf(":%s", port),
			Handler:      handler,
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
