package cerebro

import (
	"context"
	"log"
	"time"
)

type Worker struct {
	extractor *Extractor
	enricher  *Enricher
	interval  time.Duration
}

func NewWorker(extractor *Extractor, enricher *Enricher, interval time.Duration) *Worker {
	return &Worker{extractor: extractor, enricher: enricher, interval: interval}
}

func (w *Worker) Start(ctx context.Context) {
	// Run once at startup after a brief delay
	select {
	case <-ctx.Done():
		return
	case <-time.After(30 * time.Second):
	}
	w.run(ctx)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.run(ctx)
		}
	}
}

func (w *Worker) run(ctx context.Context) {
	if w.extractor == nil {
		return
	}

	// Determine extraction window
	now := time.Now()
	start := now.Add(-24 * time.Hour) // default: last 24h

	lastTime, err := w.extractor.store.GetLastExtractionTime(ctx)
	if err != nil {
		log.Printf("cerebro worker: get last extraction time: %v", err)
	} else if lastTime != nil {
		start = *lastTime
	}

	// Skip if less than 1 hour since last extraction
	if now.Sub(start) < time.Hour {
		return
	}

	if _, err := w.extractor.Extract(ctx, start, now); err != nil {
		log.Printf("cerebro worker: extraction failed: %v", err)
	}

	// Enrich top concepts needing it
	if w.enricher != nil {
		if err := w.enricher.EnrichBatch(ctx, 5); err != nil {
			log.Printf("cerebro worker: enrichment failed: %v", err)
		}
	}

	// Cleanup expired enrichments
	if deleted, err := w.extractor.store.DeleteExpiredEnrichments(ctx); err != nil {
		log.Printf("cerebro worker: cleanup failed: %v", err)
	} else if deleted > 0 {
		log.Printf("cerebro worker: cleaned up %d expired enrichments", deleted)
	}
}
