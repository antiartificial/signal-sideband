package extract

import (
	"net/url"
	"regexp"
)

var urlRegex = regexp.MustCompile(`https?://[^\s<>"{}|\\^` + "`" + `\[\]]+`)

type ExtractedURL struct {
	URL    string
	Domain string
}

func URLs(text string) []ExtractedURL {
	matches := urlRegex.FindAllString(text, -1)
	var results []ExtractedURL
	seen := make(map[string]bool)

	for _, match := range matches {
		if seen[match] {
			continue
		}
		seen[match] = true

		parsed, err := url.Parse(match)
		if err != nil {
			continue
		}

		results = append(results, ExtractedURL{
			URL:    match,
			Domain: parsed.Hostname(),
		})
	}
	return results
}
