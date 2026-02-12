package extract

import (
	"context"
	"log"
	"time"

	"signal-sideband/pkg/store"
)

type PreviewWorker struct {
	store    *store.Store
	interval time.Duration
}

func NewPreviewWorker(s *store.Store, interval time.Duration) *PreviewWorker {
	return &PreviewWorker{store: s, interval: interval}
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
		preview, err := FetchLinkPreview(u.URL)
		if err != nil {
			log.Printf("Preview worker: preview %s failed: %v", u.URL, err)
			// Mark as fetched anyway to avoid retrying endlessly
			_ = w.store.MarkURLFetched(ctx, u.ID, "", "", "")
			continue
		}

		if err := w.store.MarkURLFetched(ctx, u.ID, preview.Title, preview.Description, preview.ImageURL); err != nil {
			log.Printf("Preview worker: update %s failed: %v", u.ID, err)
			continue
		}
		log.Printf("Preview worker: fetched preview for %s: %s", u.URL, preview.Title)
	}
}
