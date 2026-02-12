package media

import (
	"context"
	"log"
	"time"

	"signal-sideband/pkg/store"
)

type Worker struct {
	store      *store.Store
	downloader *Downloader
	interval   time.Duration
}

func NewWorker(s *store.Store, d *Downloader, interval time.Duration) *Worker {
	return &Worker{store: s, downloader: d, interval: interval}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("Media worker started")
	// Run once immediately
	w.process(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Media worker stopped")
			return
		case <-ticker.C:
			w.process(ctx)
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	attachments, err := w.store.GetUndownloadedAttachments(ctx)
	if err != nil {
		log.Printf("Media worker: fetch error: %v", err)
		return
	}

	for _, a := range attachments {
		localPath, err := w.downloader.Download(a.SignalAttachmentID, a.ContentType, a.Filename)
		if err != nil {
			log.Printf("Media worker: download %s failed: %v", a.SignalAttachmentID, err)
			continue
		}

		if err := w.store.MarkAttachmentDownloaded(ctx, a.ID, localPath); err != nil {
			log.Printf("Media worker: mark downloaded %s failed: %v", a.ID, err)
			continue
		}
		log.Printf("Media worker: downloaded %s -> %s", a.SignalAttachmentID, localPath)
	}
}
