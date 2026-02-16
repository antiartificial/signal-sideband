package store

import (
	"context"
	"fmt"
	"log"
)

// truncateSample limits a message sample for display.
func truncateSample(s string, max int) string {
	if len(s) <= max {
		return s
	}
	// Cut at last space before max to avoid mid-word truncation
	for i := max; i > max-20 && i > 0; i-- {
		if s[i] == ' ' {
			return s[:i] + "..."
		}
	}
	return s[:max] + "..."
}

// GetSuperlatives computes fun stats from the last 30 days of messages.
func (s *Store) GetSuperlatives(ctx context.Context) []Superlative {
	var results []Superlative

	// 1. The Novelist — longest single message
	var novelistSender string
	var novelistLen int
	var novelistContent string
	err := s.pool.QueryRow(ctx, `
		SELECT sender_id, LENGTH(content) as len, content
		FROM messages
		WHERE LENGTH(content) > 0 AND created_at > NOW() - INTERVAL '30 days'
		AND (expires_at IS NULL OR expires_at > now())
		ORDER BY LENGTH(content) DESC LIMIT 1
	`).Scan(&novelistSender, &novelistLen, &novelistContent)
	if err == nil {
		results = append(results, Superlative{
			Label:  "The Novelist",
			Icon:   "fa-book-open",
			Winner: novelistSender,
			Value:  fmt.Sprintf("%d chars", novelistLen),
			Sample: truncateSample(novelistContent, 120),
		})
	}

	// 2. The Chatterbox — most messages
	var chatterSender string
	var chatterCount int
	err = s.pool.QueryRow(ctx, `
		SELECT sender_id, COUNT(*) as cnt
		FROM messages
		WHERE created_at > NOW() - INTERVAL '30 days'
		AND (expires_at IS NULL OR expires_at > now())
		GROUP BY sender_id ORDER BY cnt DESC LIMIT 1
	`).Scan(&chatterSender, &chatterCount)
	if err == nil {
		// Get a recent message from this sender as a sample
		var sample string
		sErr := s.pool.QueryRow(ctx, `
			SELECT content FROM messages
			WHERE sender_id = $1 AND content != ''
			AND created_at > NOW() - INTERVAL '30 days'
			AND (expires_at IS NULL OR expires_at > now())
			ORDER BY created_at DESC LIMIT 1
		`, chatterSender).Scan(&sample)
		if sErr != nil {
			sample = ""
		}
		results = append(results, Superlative{
			Label:  "The Chatterbox",
			Icon:   "fa-comments",
			Winner: chatterSender,
			Value:  fmt.Sprintf("%d messages", chatterCount),
			Sample: truncateSample(sample, 120),
		})
	}

	// 3. The Shutterbug — most media shared
	var shutterSender string
	var shutterCount int
	err = s.pool.QueryRow(ctx, `
		SELECT m.sender_id, COUNT(*) as cnt
		FROM attachments a JOIN messages m ON a.message_id = m.id
		WHERE m.created_at > NOW() - INTERVAL '30 days'
		AND (m.expires_at IS NULL OR m.expires_at > now())
		GROUP BY m.sender_id ORDER BY cnt DESC LIMIT 1
	`).Scan(&shutterSender, &shutterCount)
	if err == nil {
		// Get a sample message that had an attachment
		var sample string
		sErr := s.pool.QueryRow(ctx, `
			SELECT COALESCE(NULLIF(m.content, ''), a.filename)
			FROM attachments a JOIN messages m ON a.message_id = m.id
			WHERE m.sender_id = $1 AND m.created_at > NOW() - INTERVAL '30 days'
			AND (m.expires_at IS NULL OR m.expires_at > now())
			ORDER BY m.created_at DESC LIMIT 1
		`, shutterSender).Scan(&sample)
		if sErr != nil {
			sample = ""
		}
		results = append(results, Superlative{
			Label:  "The Shutterbug",
			Icon:   "fa-image",
			Winner: shutterSender,
			Value:  fmt.Sprintf("%d attachments", shutterCount),
			Sample: truncateSample(sample, 120),
		})
	}

	// 4. The Screamer — highest uppercase ratio (min 5 messages, min 10 char content)
	var screamerSender string
	var screamerRatio float64
	err = s.pool.QueryRow(ctx, `
		SELECT sender_id,
			SUM(LENGTH(REGEXP_REPLACE(content, '[^A-Z]', '', 'g')))::float /
			GREATEST(SUM(LENGTH(content)), 1) as caps_ratio
		FROM messages
		WHERE LENGTH(content) > 10
		AND created_at > NOW() - INTERVAL '30 days'
		AND (expires_at IS NULL OR expires_at > now())
		GROUP BY sender_id
		HAVING COUNT(*) > 5
		ORDER BY caps_ratio DESC LIMIT 1
	`).Scan(&screamerSender, &screamerRatio)
	if err == nil {
		// Get a sample high-caps message
		var sample string
		sErr := s.pool.QueryRow(ctx, `
			SELECT content FROM messages
			WHERE sender_id = $1 AND LENGTH(content) > 5
			AND created_at > NOW() - INTERVAL '30 days'
			AND (expires_at IS NULL OR expires_at > now())
			ORDER BY LENGTH(REGEXP_REPLACE(content, '[^A-Z]', '', 'g'))::float / GREATEST(LENGTH(content), 1) DESC
			LIMIT 1
		`, screamerSender).Scan(&sample)
		if sErr != nil {
			sample = ""
		}
		results = append(results, Superlative{
			Label:  "The Screamer",
			Icon:   "fa-bell-ring",
			Winner: screamerSender,
			Value:  fmt.Sprintf("%.0f%% CAPS", screamerRatio*100),
			Sample: truncateSample(sample, 120),
		})
	}

	// 5. The Minimalist — shortest average message length (min 10 messages)
	var minSender string
	var minAvg int
	err = s.pool.QueryRow(ctx, `
		SELECT sender_id, AVG(LENGTH(content))::int as avg_len
		FROM messages
		WHERE content != ''
		AND created_at > NOW() - INTERVAL '30 days'
		AND (expires_at IS NULL OR expires_at > now())
		GROUP BY sender_id
		HAVING COUNT(*) > 10
		ORDER BY avg_len ASC LIMIT 1
	`).Scan(&minSender, &minAvg)
	if err == nil {
		// Get a typical short message
		var sample string
		sErr := s.pool.QueryRow(ctx, `
			SELECT content FROM messages
			WHERE sender_id = $1 AND content != ''
			AND created_at > NOW() - INTERVAL '30 days'
			AND (expires_at IS NULL OR expires_at > now())
			ORDER BY LENGTH(content) ASC LIMIT 1
		`, minSender).Scan(&sample)
		if sErr != nil {
			sample = ""
		}
		results = append(results, Superlative{
			Label:  "The Minimalist",
			Icon:   "fa-compress",
			Winner: minSender,
			Value:  fmt.Sprintf("avg %d chars", minAvg),
			Sample: truncateSample(sample, 120),
		})
	}

	// 6. The Marathon — longest streak of consecutive messages
	var streakSender string
	var streakLen int
	err = s.pool.QueryRow(ctx, `
		WITH ranked AS (
			SELECT sender_id,
				ROW_NUMBER() OVER (ORDER BY created_at) -
				ROW_NUMBER() OVER (PARTITION BY sender_id ORDER BY created_at) AS grp
			FROM messages
			WHERE created_at > NOW() - INTERVAL '30 days'
			AND (expires_at IS NULL OR expires_at > now())
		),
		streaks AS (
			SELECT sender_id, COUNT(*) as streak_len
			FROM ranked
			GROUP BY sender_id, grp
		)
		SELECT sender_id, MAX(streak_len) as longest
		FROM streaks
		GROUP BY sender_id
		ORDER BY longest DESC LIMIT 1
	`).Scan(&streakSender, &streakLen)
	if err == nil && streakLen > 1 {
		// Get a sample from this sender
		var sample string
		sErr := s.pool.QueryRow(ctx, `
			SELECT content FROM messages
			WHERE sender_id = $1 AND content != ''
			AND created_at > NOW() - INTERVAL '30 days'
			AND (expires_at IS NULL OR expires_at > now())
			ORDER BY created_at DESC LIMIT 1
		`, streakSender).Scan(&sample)
		if sErr != nil {
			sample = ""
		}
		results = append(results, Superlative{
			Label:  "The Marathon",
			Icon:   "fa-bolt",
			Winner: streakSender,
			Value:  fmt.Sprintf("%d in a row", streakLen),
			Sample: truncateSample(sample, 120),
		})
	}

	// 7. The Curator — most links shared
	var curatorSender string
	var curatorCount int
	err = s.pool.QueryRow(ctx, `
		SELECT m.sender_id, COUNT(*) as cnt
		FROM urls u JOIN messages m ON u.message_id = m.id
		WHERE m.created_at > NOW() - INTERVAL '30 days'
		AND (m.expires_at IS NULL OR m.expires_at > now())
		GROUP BY m.sender_id ORDER BY cnt DESC LIMIT 1
	`).Scan(&curatorSender, &curatorCount)
	if err == nil {
		// Get a sample link title or URL
		var sample string
		sErr := s.pool.QueryRow(ctx, `
			SELECT COALESCE(NULLIF(u.title, ''), u.url)
			FROM urls u JOIN messages m ON u.message_id = m.id
			WHERE m.sender_id = $1 AND m.created_at > NOW() - INTERVAL '30 days'
			AND (m.expires_at IS NULL OR m.expires_at > now())
			ORDER BY m.created_at DESC LIMIT 1
		`, curatorSender).Scan(&sample)
		if sErr != nil {
			sample = ""
		}
		results = append(results, Superlative{
			Label:  "The Curator",
			Icon:   "fa-link",
			Winner: curatorSender,
			Value:  fmt.Sprintf("%d links", curatorCount),
			Sample: truncateSample(sample, 120),
		})
	}

	// 8. The Director — most videos shared
	var directorSender string
	var directorCount int
	err = s.pool.QueryRow(ctx, `
		SELECT m.sender_id, COUNT(*) as cnt
		FROM attachments a JOIN messages m ON a.message_id = m.id
		WHERE a.content_type LIKE 'video/%'
		AND m.created_at > NOW() - INTERVAL '30 days'
		AND (m.expires_at IS NULL OR m.expires_at > now())
		GROUP BY m.sender_id ORDER BY cnt DESC LIMIT 1
	`).Scan(&directorSender, &directorCount)
	if err == nil {
		// Get a sample video message/filename
		var sample string
		sErr := s.pool.QueryRow(ctx, `
			SELECT COALESCE(NULLIF(m.content, ''), a.filename)
			FROM attachments a JOIN messages m ON a.message_id = m.id
			WHERE m.sender_id = $1 AND a.content_type LIKE 'video/%'
			AND m.created_at > NOW() - INTERVAL '30 days'
			AND (m.expires_at IS NULL OR m.expires_at > now())
			ORDER BY m.created_at DESC LIMIT 1
		`, directorSender).Scan(&sample)
		if sErr != nil {
			sample = ""
		}
		results = append(results, Superlative{
			Label:  "The Director",
			Icon:   "fa-film",
			Winner: directorSender,
			Value:  fmt.Sprintf("%d videos", directorCount),
			Sample: truncateSample(sample, 120),
		})
	}

	if len(results) == 0 {
		log.Println("Superlatives: no data available")
	}

	return results
}
