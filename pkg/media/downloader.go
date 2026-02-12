package media

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"signal-sideband/pkg/signal"
)

type Downloader struct {
	api       *signal.APIClient
	mediaPath string
}

func NewDownloader(api *signal.APIClient, mediaPath string) *Downloader {
	return &Downloader{api: api, mediaPath: mediaPath}
}

func (d *Downloader) Download(attachmentID, contentType, filename string) (string, error) {
	body, _, err := d.api.DownloadAttachment(attachmentID)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	defer body.Close()

	// Determine subdirectory from content type (e.g. "image/jpeg" -> "image")
	category := "other"
	if parts := strings.SplitN(contentType, "/", 2); len(parts) == 2 {
		category = parts[0]
	}

	dir := filepath.Join(d.mediaPath, category)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("mkdir: %w", err)
	}

	// Determine extension
	ext := filepath.Ext(filename)
	if ext == "" {
		ext = extFromContentType(contentType)
	}

	localPath := filepath.Join(dir, attachmentID+ext)
	f, err := os.Create(localPath)
	if err != nil {
		return "", fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	if _, err := io.Copy(f, body); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	return localPath, nil
}

func extFromContentType(ct string) string {
	switch ct {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "audio/mpeg":
		return ".mp3"
	case "audio/ogg":
		return ".ogg"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}
