package extract

import (
	"context"
	"log"
	"strings"
	"time"

	"signal-sideband/pkg/llm"
	"signal-sideband/pkg/store"
)

type PreviewWorker struct {
	store       *store.Store
	interval    time.Duration
	llmProvider llm.Provider
}

func NewPreviewWorker(s *store.Store, interval time.Duration, provider llm.Provider) *PreviewWorker {
	return &PreviewWorker{store: s, interval: interval, llmProvider: provider}
}

func (w *PreviewWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("Preview worker started")
	w.process(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Preview worker stopped")
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *PreviewWorker) process(ctx context.Context) {
	urls, err := w.store.GetUnfetchedURLs(ctx)
	if err != nil {
		log.Printf("Preview worker: fetch error: %v", err)
		return
	}

	for _, u := range urls {
		var title, description, imageURL string

		// Skip HTML scrape for X/Twitter (auth wall blocks it)
		if !isTwitterDomain(u.Domain) {
			preview, err := FetchLinkPreview(u.URL)
			if err != nil {
				log.Printf("Preview worker: preview %s failed: %v", u.URL, err)
			} else {
				title = preview.Title
				description = preview.Description
				imageURL = preview.ImageURL
			}
		}

		// LLM summarization
		var summary string
		var tags []string
		if w.llmProvider != nil {
			s, t, err := SummarizeURL(ctx, w.llmProvider, u.URL, u.Domain, title, description, u.SharedBy)
			if err != nil {
				log.Printf("Preview worker: summarize %s failed: %v", u.URL, err)
			} else {
				summary = s
				tags = t
				log.Printf("Preview worker: summarized %s: %s [%s]", u.URL, summary, strings.Join(tags, ", "))
			}
		}

		if err := w.store.MarkURLFetched(ctx, u.ID, title, description, imageURL, summary, tags); err != nil {
			log.Printf("Preview worker: update %s failed: %v", u.ID, err)
			continue
		}
		log.Printf("Preview worker: fetched preview for %s: %s", u.URL, title)
	}

	// Backfill: summarize already-fetched URLs that have no summary yet
	if w.llmProvider == nil {
		return
	}
	unsummarized, err := w.store.GetUnsummarizedURLs(ctx)
	if err != nil {
		log.Printf("Preview worker: get unsummarized error: %v", err)
		return
	}
	for _, u := range unsummarized {
		s, t, err := SummarizeURL(ctx, w.llmProvider, u.URL, u.Domain, u.Title, u.Description, u.SharedBy)
		if err != nil {
			log.Printf("Preview worker: backfill summarize %s failed: %v", u.URL, err)
			// Write empty summary to avoid retrying forever
			_ = w.store.UpdateURLSummary(ctx, u.ID, "", nil)
			continue
		}
		if err := w.store.UpdateURLSummary(ctx, u.ID, s, t); err != nil {
			log.Printf("Preview worker: backfill update %s failed: %v", u.ID, err)
			continue
		}
		log.Printf("Preview worker: backfilled %s: %s [%s]", u.URL, s, strings.Join(t, ", "))
	}
}
