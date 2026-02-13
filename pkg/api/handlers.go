package api

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"signal-sideband/pkg/ai"
	"signal-sideband/pkg/cerebro"
	"signal-sideband/pkg/digest"
	"signal-sideband/pkg/media"
	"signal-sideband/pkg/store"
)

type Handlers struct {
	store             *store.Store
	embedder          ai.Embedder
	generator         *digest.Generator
	insightsGen       *digest.InsightsGenerator
	picGen            *media.PicOfDayGenerator
	cerebroExtractor  *cerebro.Extractor
	cerebroEnricher   *cerebro.Enricher
	mediaPath         string
	authPassword      string
}

func NewHandlers(s *store.Store, e ai.Embedder, g *digest.Generator, ig *digest.InsightsGenerator, picGen *media.PicOfDayGenerator, cerebroExtractor *cerebro.Extractor, cerebroEnricher *cerebro.Enricher, mediaPath string, authPassword string) *Handlers {
	return &Handlers{store: s, embedder: e, generator: g, insightsGen: ig, picGen: picGen, cerebroExtractor: cerebroExtractor, cerebroEnricher: cerebroEnricher, mediaPath: mediaPath, authPassword: authPassword}
}

type loginRequest struct {
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *Handlers) AuthStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]bool{"required": h.authPassword != ""})
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if subtle.ConstantTimeCompare([]byte(req.Password), []byte(h.authPassword)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid password")
		return
	}

	token, err := generateToken(h.authPassword)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "token generation failed")
		return
	}

	writeJSON(w, http.StatusOK, loginResponse{Token: token})
}

func (h *Handlers) GetMessages(w http.ResponseWriter, r *http.Request) {
	filter := store.MessageFilter{
		Limit:  intParam(r, "limit", 50),
		Offset: intParam(r, "offset", 0),
	}

	if v := r.URL.Query().Get("group_id"); v != "" {
		filter.GroupID = &v
	}
	if v := r.URL.Query().Get("sender_id"); v != "" {
		filter.SenderID = &v
	}
	if v := r.URL.Query().Get("after"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.After = &t
		}
	}
	if v := r.URL.Query().Get("before"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.Before = &t
		}
	}
	if v := r.URL.Query().Get("has_media"); v == "true" {
		b := true
		filter.HasMedia = &b
	}

	messages, total, err := h.store.ListMessages(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if messages == nil {
		messages = []store.MessageRecord{}
	}
	writePaginated(w, messages, total, filter.Limit, filter.Offset)
}

func (h *Handlers) SearchMessages(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}

	mode := r.URL.Query().Get("mode")
	limit := intParam(r, "limit", 20)

	// Parse search filters
	var filter store.SearchFilter
	if v := r.URL.Query().Get("group_id"); v != "" {
		filter.GroupID = &v
	}
	if v := r.URL.Query().Get("sender_id"); v != "" {
		filter.SenderID = &v
	}
	if v := r.URL.Query().Get("after"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.After = &t
		}
	}
	if v := r.URL.Query().Get("before"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.Before = &t
		}
	}
	if r.URL.Query().Get("has_media") == "true" {
		b := true
		filter.HasMedia = &b
	}

	switch mode {
	case "semantic":
		embedding, err := h.embedder.Embed(query)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "embedding error: "+err.Error())
			return
		}
		results, err := h.store.FilteredSemanticSearch(r.Context(), embedding, 0.5, filter, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []store.SearchResult{}
		}
		writeJSON(w, http.StatusOK, results)

	default: // fulltext
		results, err := h.store.FilteredFullTextSearch(r.Context(), query, filter, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []store.SearchResult{}
		}
		writeJSON(w, http.StatusOK, results)
	}
}

func (h *Handlers) GetGroups(w http.ResponseWriter, r *http.Request) {
	groups, err := h.store.ListGroups(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if groups == nil {
		groups = []store.GroupWithCount{}
	}
	writeJSON(w, http.StatusOK, groups)
}

func (h *Handlers) GetDigests(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 20)
	offset := intParam(r, "offset", 0)

	digests, total, err := h.store.ListDigests(r.Context(), limit, offset)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if digests == nil {
		digests = []store.DigestRecord{}
	}
	writePaginated(w, digests, total, limit, offset)
}

func (h *Handlers) GetDigest(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}

	d, err := h.store.GetDigest(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "digest not found")
		return
	}
	writeJSON(w, http.StatusOK, d)
}

type generateRequest struct {
	PeriodStart string  `json:"period_start"`
	PeriodEnd   string  `json:"period_end"`
	GroupID     *string `json:"group_id,omitempty"`
	Lens        string  `json:"lens,omitempty"`
}

func (h *Handlers) GenerateDigest(w http.ResponseWriter, r *http.Request) {
	if h.generator == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM provider not configured")
		return
	}

	var req generateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	start, err := time.Parse(time.RFC3339, req.PeriodStart)
	if err != nil {
		// Try date-only format
		start, err = time.Parse("2006-01-02", req.PeriodStart)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid period_start format (use RFC3339 or YYYY-MM-DD)")
			return
		}
	}

	end, err := time.Parse(time.RFC3339, req.PeriodEnd)
	if err != nil {
		end, err = time.Parse("2006-01-02", req.PeriodEnd)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid period_end format (use RFC3339 or YYYY-MM-DD)")
			return
		}
		// Set to end of day
		end = end.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
	}

	d, err := h.generator.Generate(r.Context(), start, end, req.GroupID, req.Lens)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, d)
}

func (h *Handlers) GetURLs(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 50)
	offset := intParam(r, "offset", 0)
	var domain *string
	if v := r.URL.Query().Get("domain"); v != "" {
		domain = &v
	}

	urls, total, err := h.store.ListURLs(r.Context(), limit, offset, domain)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if urls == nil {
		urls = []store.URLRecord{}
	}
	writePaginated(w, urls, total, limit, offset)
}

func (h *Handlers) ServeMedia(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}

	attachment, err := h.store.GetAttachment(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}

	if !attachment.Downloaded || attachment.LocalPath == "" {
		writeError(w, http.StatusNotFound, "attachment not yet downloaded")
		return
	}

	if _, err := os.Stat(attachment.LocalPath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "file not found on disk")
		return
	}

	w.Header().Set("Content-Type", attachment.ContentType)
	http.ServeFile(w, r, attachment.LocalPath)
}

func (h *Handlers) GetMedia(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 50)
	offset := intParam(r, "offset", 0)
	sort := r.URL.Query().Get("sort")

	attachments, total, err := h.store.ListAllAttachments(r.Context(), limit, offset, sort)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if attachments == nil {
		attachments = []store.AttachmentRecord{}
	}
	writePaginated(w, attachments, total, limit, offset)
}

func (h *Handlers) GetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.store.GetStats(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (h *Handlers) ServeMediaThumb(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}

	attachment, err := h.store.GetAttachment(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "attachment not found")
		return
	}

	// Serve thumbnail if available
	if attachment.ThumbnailPath != "" {
		if _, err := os.Stat(attachment.ThumbnailPath); err == nil {
			w.Header().Set("Content-Type", "image/jpeg")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			http.ServeFile(w, r, attachment.ThumbnailPath)
			return
		}
	}

	// Fallback: serve original for images
	if attachment.Downloaded && attachment.LocalPath != "" {
		if _, err := os.Stat(attachment.LocalPath); err == nil {
			w.Header().Set("Content-Type", attachment.ContentType)
			w.Header().Set("Cache-Control", "public, max-age=86400")
			http.ServeFile(w, r, attachment.LocalPath)
			return
		}
	}

	writeError(w, http.StatusNotFound, "thumbnail not available")
}

func (h *Handlers) SearchMedia(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query().Get("q")
	if query == "" {
		writeError(w, http.StatusBadRequest, "query parameter 'q' is required")
		return
	}
	limit := intParam(r, "limit", 50)

	results, err := h.store.SearchMedia(r.Context(), query, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if results == nil {
		results = []store.MediaSearchResult{}
	}
	writeJSON(w, http.StatusOK, results)
}

func (h *Handlers) ServePicOfDay(w http.ResponseWriter, r *http.Request) {
	// Try latest insight first, then fall back to most recent with an image
	imagePath, err := h.store.GetLatestPicOfDay(r.Context())
	if err != nil || imagePath == "" {
		writeError(w, http.StatusNotFound, "no picture of the day available")
		return
	}

	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		writeError(w, http.StatusNotFound, "image file not found")
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, imagePath)
}

func (h *Handlers) GenerateInsight(w http.ResponseWriter, r *http.Request) {
	if h.insightsGen == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM provider not configured")
		return
	}

	if err := h.insightsGen.GenerateDailyInsights(r.Context()); err != nil {
		writeError(w, http.StatusInternalServerError, "insight generation failed: "+err.Error())
		return
	}

	insight, err := h.store.GetLatestInsight(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to retrieve generated insight")
		return
	}

	writeJSON(w, http.StatusOK, insight)
}

func (h *Handlers) GeneratePicOfDay(w http.ResponseWriter, r *http.Request) {
	if h.picGen == nil {
		writeError(w, http.StatusServiceUnavailable, "GEMINI_API_KEY not configured")
		return
	}

	// Get latest insight for themes
	insight, err := h.store.GetLatestInsight(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no daily insight available — generate one first")
		return
	}

	var themes []string
	_ = json.Unmarshal(insight.Themes, &themes)
	if len(themes) == 0 {
		themes = []string{"technology", "conversation", "daily life"}
	}

	imagePath, err := h.picGen.Generate(r.Context(), themes, insight.Overview)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "image generation failed: "+err.Error())
		return
	}

	// Save the image path to the insight
	if err := h.store.SetInsightImagePath(r.Context(), insight.ID, imagePath); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to save image path: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{
		"image_path": imagePath,
		"insight_id": insight.ID,
	})
}

func intParam(r *http.Request, key string, fallback int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 0 {
		return fallback
	}
	return n
}

// Cerebro handlers

func (h *Handlers) GetCerebroGraph(w http.ResponseWriter, r *http.Request) {
	limit := intParam(r, "limit", 50)
	var groupID *string
	if v := r.URL.Query().Get("group_id"); v != "" {
		groupID = &v
	}
	var since *time.Time
	if v := r.URL.Query().Get("since"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			since = &t
		}
	}

	graph, err := h.store.GetCerebroGraph(r.Context(), groupID, since, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, graph)
}

func (h *Handlers) GetCerebroConcept(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}

	detail, err := h.store.GetConceptDetail(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "concept not found")
		return
	}
	writeJSON(w, http.StatusOK, detail)
}

func (h *Handlers) EnrichCerebroConcept(w http.ResponseWriter, r *http.Request) {
	if h.cerebroEnricher == nil {
		writeError(w, http.StatusServiceUnavailable, "enrichment providers not configured")
		return
	}

	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "id required")
		return
	}

	detail, err := h.store.GetConceptDetail(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "concept not found")
		return
	}

	if err := h.cerebroEnricher.EnrichConcept(r.Context(), detail.CerebroConcept); err != nil {
		writeError(w, http.StatusInternalServerError, "enrichment failed: "+err.Error())
		return
	}

	// Re-fetch with enrichments
	updated, err := h.store.GetConceptDetail(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

func (h *Handlers) ExtractCerebro(w http.ResponseWriter, r *http.Request) {
	if h.cerebroExtractor == nil {
		writeError(w, http.StatusServiceUnavailable, "LLM provider not configured")
		return
	}

	end := time.Now()
	start := end.Add(-24 * time.Hour)

	extraction, err := h.cerebroExtractor.Extract(r.Context(), start, end)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	writeJSON(w, http.StatusOK, extraction)
}
