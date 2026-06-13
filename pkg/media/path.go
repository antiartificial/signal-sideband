package media

import (
	"path/filepath"
	"strings"
)

func ResolveMediaPath(path, mediaPath string) string {
	if path == "" || filepath.IsAbs(path) || !strings.HasPrefix(path, "media/") {
		return path
	}
	if mediaPath == "" || mediaPath == "./media" {
		return path
	}
	return filepath.Join(mediaPath, strings.TrimPrefix(path, "media/"))
}
