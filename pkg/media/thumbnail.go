package media

import (
	"context"
	"fmt"
	"image"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const thumbMaxDim = 300
const ffmpegTimeout = 45 * time.Second

func IsVisualMedia(contentType string) bool {
	return strings.HasPrefix(contentType, "image/") || strings.HasPrefix(contentType, "video/")
}

func GenerateThumbnail(localPath, contentType, attachmentID, mediaPath string) (string, error) {
	thumbDir := filepath.Join(mediaPath, "thumbs")
	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		return "", fmt.Errorf("mkdir thumbs: %w", err)
	}

	outPath := filepath.Join(thumbDir, attachmentID+".jpg")

	if strings.HasPrefix(contentType, "video/") {
		return generateVideoThumbnail(localPath, outPath)
	}
	return generateImageThumbnail(localPath, outPath)
}

func generateImageThumbnail(srcPath, outPath string) (string, error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return "", fmt.Errorf("open source: %w", err)
	}
	defer f.Close()

	src, _, err := image.Decode(f)
	if err != nil {
		return "", fmt.Errorf("decode image: %w", err)
	}

	dst := resizeImage(src)

	out, err := os.Create(outPath)
	if err != nil {
		return "", fmt.Errorf("create thumb: %w", err)
	}
	defer out.Close()

	if err := jpeg.Encode(out, dst, &jpeg.Options{Quality: 80}); err != nil {
		return "", fmt.Errorf("encode jpeg: %w", err)
	}
	return outPath, nil
}

func generateVideoThumbnail(srcPath, outPath string) (string, error) {
	tmpFrame := outPath + ".tmp.jpg"
	defer os.Remove(tmpFrame)

	// Extract and scale in ffmpeg so Go only decodes a small JPEG frame.
	if err := runFFmpegFrame(srcPath, tmpFrame, "1"); err != nil {
		// Try at 0 seconds if 1s fails (very short videos)
		if err := runFFmpegFrame(srcPath, tmpFrame, "0"); err != nil {
			return "", fmt.Errorf("ffmpeg frame extract: %w", err)
		}
	}

	// Resize the extracted frame
	return generateImageThumbnail(tmpFrame, outPath)
}

func runFFmpegFrame(srcPath, outPath, offset string) error {
	ctx, cancel := context.WithTimeout(context.Background(), ffmpegTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-v", "error",
		"-nostdin",
		"-threads", "1",
		"-ss", offset,
		"-i", srcPath,
		"-frames:v", "1",
		"-vf", "scale='min(300,iw)':-2",
		"-q:v", "3",
		"-y",
		outPath,
	)
	cmd.WaitDelay = 2 * time.Second
	if err := cmd.Run(); err != nil {
		return err
	}
	if info, err := os.Stat(outPath); err != nil {
		return fmt.Errorf("ffmpeg produced no frame: %w", err)
	} else if info.Size() == 0 {
		return fmt.Errorf("ffmpeg produced empty frame")
	}
	return nil
}

func resizeImage(src image.Image) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	if w <= thumbMaxDim && h <= thumbMaxDim {
		return src
	}

	var newW, newH int
	if w > h {
		newW = thumbMaxDim
		newH = h * thumbMaxDim / w
	} else {
		newH = thumbMaxDim
		newW = w * thumbMaxDim / h
	}
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.BiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)
	return dst
}
