package media

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PicOfDayGenerator generates a daily "nano banana" themed image via Gemini.
type PicOfDayGenerator struct {
	apiKey    string
	mediaPath string
	model     string
}

func NewPicOfDayGenerator(apiKey, mediaPath string) *PicOfDayGenerator {
	return &PicOfDayGenerator{
		apiKey:    apiKey,
		mediaPath: mediaPath,
		model:     "gemini-2.5-flash-image",
	}
}

type geminiRequest struct {
	Contents         []geminiContent  `json:"contents"`
	GenerationConfig geminiGenConfig  `json:"generationConfig"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text       string          `json:"text,omitempty"`
	InlineData *geminiInline   `json:"inlineData,omitempty"`
}

type geminiGenConfig struct {
	ResponseModalities []string `json:"responseModalities"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text       string `json:"text,omitempty"`
				InlineData *struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData,omitempty"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
		Code    int    `json:"code"`
	} `json:"error,omitempty"`
}

type geminiInline struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

// Generate creates a nano banana picture of the day based on themes and overview.
// Returns the path to the saved image file.
func (g *PicOfDayGenerator) Generate(ctx context.Context, themes []string, overview string) (string, error) {
	// Build the prompt
	themeList := strings.Join(themes, ", ")
	prompt := fmt.Sprintf(
		`Generate a whimsical illustration of a tiny, adorable golden banana character called "nano banana." `+
			`Today's conversation themes were: %s. `+
			`Context: %s `+
			`The nano banana should be doing something playful that relates to these themes. `+
			`Style: cute kawaii-inspired digital illustration, warm vibrant colors, clean composition, `+
			`slightly surreal, the banana has tiny arms and legs and an expressive face. `+
			`No text or words in the image.`,
		themeList, truncateStr(overview, 200),
	)

	log.Printf("PicOfDay: generating with themes [%s]", themeList)

	// Call Gemini API
	reqBody := geminiRequest{
		Contents: []geminiContent{
			{Parts: []geminiPart{{Text: prompt}}},
		},
		GenerationConfig: geminiGenConfig{
			ResponseModalities: []string{"TEXT", "IMAGE"},
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf(
		"https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s",
		g.model, g.apiKey,
	)

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("gemini request: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("gemini API error (status %d): %s", resp.StatusCode, string(respBytes))
	}

	var gemResp geminiResponse
	if err := json.Unmarshal(respBytes, &gemResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if gemResp.Error != nil {
		return "", fmt.Errorf("gemini error: %s", gemResp.Error.Message)
	}

	// Extract the image from the response
	var imageData []byte
	var mimeType string
	for _, candidate := range gemResp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.InlineData != nil {
				data, err := base64.StdEncoding.DecodeString(part.InlineData.Data)
				if err != nil {
					return "", fmt.Errorf("decode image: %w", err)
				}
				imageData = data
				mimeType = part.InlineData.MimeType
				break
			}
		}
		if imageData != nil {
			break
		}
	}

	if imageData == nil {
		return "", fmt.Errorf("no image in gemini response")
	}

	// Determine file extension
	ext := ".png"
	if strings.Contains(mimeType, "jpeg") || strings.Contains(mimeType, "jpg") {
		ext = ".jpg"
	} else if strings.Contains(mimeType, "webp") {
		ext = ".webp"
	}

	// Save to disk
	potdDir := filepath.Join(g.mediaPath, "potd")
	if err := os.MkdirAll(potdDir, 0755); err != nil {
		return "", fmt.Errorf("create potd dir: %w", err)
	}

	filename := fmt.Sprintf("nano-banana-%s%s", time.Now().Format("2006-01-02"), ext)
	outPath := filepath.Join(potdDir, filename)

	if err := os.WriteFile(outPath, imageData, 0644); err != nil {
		return "", fmt.Errorf("write image: %w", err)
	}

	log.Printf("PicOfDay: saved %s (%d bytes)", outPath, len(imageData))
	return outPath, nil
}

func truncateStr(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
