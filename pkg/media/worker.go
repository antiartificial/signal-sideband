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
	mediaPath  string
}

func NewWorker(s *store.Store, d *Downloader, interval time.Duration, mediaPath string) *Worker {
	return &Worker{store: s, downloader: d, interval: interval, mediaPath: mediaPath}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	log.Println("Media worker started")
	// Run once immediately
	w.process(ctx)
	w.backfillThumbnails(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Media worker stopped")
			return
		case <-ticker.C:
			w.process(ctx)
			w.backfillThumbnails(ctx)
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

		// Generate thumbnail for visual media
		if IsVisualMedia(a.ContentType) {
			thumbPath, err := GenerateThumbnail(localPath, a.ContentType, a.ID, w.mediaPath)
			if err != nil {
				log.Printf("Media worker: thumbnail %s failed: %v", a.ID, err)
				continue
			}
			if err := w.store.SetThumbnailPath(ctx, a.ID, thumbPath); err != nil {
				log.Printf("Media worker: save thumbnail path %s failed: %v", a.ID, err)
			} else {
				log.Printf("Media worker: thumbnail %s -> %s", a.ID, thumbPath)
			}
		}
	}
}

func (w *Worker) backfillThumbnails(ctx context.Context) {
	attachments, err := w.store.GetUnthumbnailedAttachments(ctx)
	if err != nil {
		log.Printf("Media worker: backfill fetch error: %v", err)
		return
	}

	for _, a := range attachments {
		if a.LocalPath == "" {
			continue
		}
		thumbPath, err := GenerateThumbnail(a.LocalPath, a.ContentType, a.ID, w.mediaPath)
		if err != nil {
			log.Printf("Media worker: backfill thumbnail %s failed: %v", a.ID, err)
			continue
		}
		if err := w.store.SetThumbnailPath(ctx, a.ID, thumbPath); err != nil {
			log.Printf("Media worker: backfill save %s failed: %v", a.ID, err)
		} else {
			log.Printf("Media worker: backfill thumbnail %s -> %s", a.ID, thumbPath)
		}
	}
}
