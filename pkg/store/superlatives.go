package store

import (
	"context"
	"fmt"
	"log"
)

// GetSuperlatives computes fun stats from the last 30 days of messages.
func (s *Store) GetSuperlatives(ctx context.Context) []Superlative {
	var results []Superlative

	type query struct {
		label string
		icon  string
		sql   string
		scan  func(dest ...any) error
	}

	// 1. The Novelist — longest single message
	var novelistSender string
	var novelistLen int
	err := s.pool.QueryRow(ctx, `
		SELECT sender_id, LENGTH(content) as len
		FROM messages
		WHERE LENGTH(content) > 0 AND created_at > NOW() - INTERVAL '30 days'
		AND (expires_at IS NULL OR expires_at > now())
		ORDER BY LENGTH(content) DESC LIMIT 1
	`).Scan(&novelistSender, &novelistLen)
	if err == nil {
		results = append(results, Superlative{
			Label:  "The Novelist",
			Icon:   "fa-book-open",
			Winner: novelistSender,
			Value:  fmt.Sprintf("%d chars", novelistLen),
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
		results = append(results, Superlative{
			Label:  "The Chatterbox",
			Icon:   "fa-comments",
			Winner: chatterSender,
			Value:  fmt.Sprintf("%d messages", chatterCount),
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
		results = append(results, Superlative{
			Label:  "The Shutterbug",
			Icon:   "fa-image",
			Winner: shutterSender,
			Value:  fmt.Sprintf("%d attachments", shutterCount),
		})
	}

	// 4. The Screamer — highest uppercase ratio (min 10 messages, min 10 char avg)
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
		results = append(results, Superlative{
			Label:  "The Screamer",
			Icon:   "fa-bell-ring",
			Winner: screamerSender,
			Value:  fmt.Sprintf("%.0f%% CAPS", screamerRatio*100),
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
		results = append(results, Superlative{
			Label:  "The Minimalist",
			Icon:   "fa-compress",
			Winner: minSender,
			Value:  fmt.Sprintf("avg %d chars", minAvg),
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
		results = append(results, Superlative{
			Label:  "The Marathon",
			Icon:   "fa-bolt",
			Winner: streakSender,
			Value:  fmt.Sprintf("%d in a row", streakLen),
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
		results = append(results, Superlative{
			Label:  "The Curator",
			Icon:   "fa-link",
			Winner: curatorSender,
			Value:  fmt.Sprintf("%d links", curatorCount),
		})
	}

	if len(results) == 0 {
		log.Println("Superlatives: no data available")
	}

	return results
}
