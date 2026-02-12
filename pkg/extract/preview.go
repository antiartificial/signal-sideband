package extract

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

type LinkPreview struct {
	Title       string
	Description string
	ImageURL    string
}

var (
	titleRegex       = regexp.MustCompile(`<title[^>]*>(.*?)</title>`)
	ogTitleRegex     = regexp.MustCompile(`<meta[^>]+property=["']og:title["'][^>]+content=["']([^"']+)["']`)
	ogDescRegex      = regexp.MustCompile(`<meta[^>]+property=["']og:description["'][^>]+content=["']([^"']+)["']`)
	ogImageRegex     = regexp.MustCompile(`<meta[^>]+property=["']og:image["'][^>]+content=["']([^"']+)["']`)
	metaDescRegex    = regexp.MustCompile(`<meta[^>]+name=["']description["'][^>]+content=["']([^"']+)["']`)
	// Also match reversed attribute order (content before property/name)
	ogTitleRegexAlt  = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:title["']`)
	ogDescRegexAlt   = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:description["']`)
	ogImageRegexAlt  = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+property=["']og:image["']`)
	metaDescRegexAlt = regexp.MustCompile(`<meta[^>]+content=["']([^"']+)["'][^>]+name=["']description["']`)
)

func FetchLinkPreview(rawURL string) (*LinkPreview, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 5 {
				return fmt.Errorf("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; SignalSideband/1.0)")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	// Read limited body (first 64KB should contain all meta tags)
	limited := io.LimitReader(resp.Body, 64*1024)
	body, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	html := string(body)

	preview := &LinkPreview{}

	// Try OG tags first, fall back to standard HTML tags
	preview.Title = firstMatch(html, ogTitleRegex, ogTitleRegexAlt, titleRegex)
	preview.Description = firstMatch(html, ogDescRegex, ogDescRegexAlt, metaDescRegex, metaDescRegexAlt)
	preview.ImageURL = firstMatch(html, ogImageRegex, ogImageRegexAlt)

	preview.Title = strings.TrimSpace(preview.Title)
	preview.Description = strings.TrimSpace(preview.Description)

	return preview, nil
}

func firstMatch(html string, patterns ...*regexp.Regexp) string {
	for _, re := range patterns {
		if m := re.FindStringSubmatch(html); len(m) > 1 {
			return m[1]
		}
	}
	return ""
}
