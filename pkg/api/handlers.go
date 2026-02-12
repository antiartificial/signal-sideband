package api

import (
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"time"

	"signal-sideband/pkg/ai"
	"signal-sideband/pkg/digest"
	"signal-sideband/pkg/store"
)

type Handlers struct {
	store     *store.Store
	embedder  ai.Embedder
	generator *digest.Generator
	mediaPath string
}

func NewHandlers(s *store.Store, e ai.Embedder, g *digest.Generator, mediaPath string) *Handlers {
	return &Handlers{store: s, embedder: e, generator: g, mediaPath: mediaPath}
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

	switch mode {
	case "semantic":
		embedding, err := h.embedder.Embed(query)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "embedding error: "+err.Error())
			return
		}
		results, err := h.store.SemanticSearch(r.Context(), embedding, 0.5, limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if results == nil {
			results = []store.SearchResult{}
		}
		writeJSON(w, http.StatusOK, results)

	default: // fulltext
		results, err := h.store.FullTextSearch(r.Context(), query, limit)
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

	d, err := h.generator.Generate(r.Context(), start, end, req.GroupID)
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

	attachments, total, err := h.store.ListAllAttachments(r.Context(), limit, offset)
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
