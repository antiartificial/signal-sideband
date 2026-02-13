package store

import (
	"context"
	"encoding/json"
	"time"
)

func (s *Store) ComputeDaySnapshot(ctx context.Context, date time.Time) (*DaySnapshot, error) {
	loc := date.Location()
	dayStart := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, loc)
	dayEnd := dayStart.Add(24 * time.Hour)

	snap := &DaySnapshot{
		Crews: make(map[string][]CrewMember),
	}

	// 1. Message count + active senders
	err := s.pool.QueryRow(ctx, `
		SELECT COUNT(*), COUNT(DISTINCT sender_id)
		FROM messages
		WHERE created_at >= $1 AND created_at < $2
		AND (expires_at IS NULL OR expires_at > now())
	`, dayStart, dayEnd).Scan(&snap.MessageCount, &snap.ActiveSenders)
	if err != nil {
		return nil, err
	}

	if snap.MessageCount == 0 {
		return snap, nil
	}

	// 2. Busiest hour
	_ = s.pool.QueryRow(ctx, `
		SELECT EXTRACT(HOUR FROM created_at)::int
		FROM messages
		WHERE created_at >= $1 AND created_at < $2
		AND (expires_at IS NULL OR expires_at > now())
		GROUP BY 1
		ORDER BY COUNT(*) DESC
		LIMIT 1
	`, dayStart, dayEnd).Scan(&snap.BusiestHour)

	// 3. Time-of-day crews
	type crewRange struct {
		name      string
		hourStart int
		hourEnd   int
	}
	crews := []crewRange{
		{"morning", 6, 12},
		{"afternoon", 12, 17},
		{"evening", 17, 22},
		{"night_a", 22, 24}, // split night: 22-24 and 0-6
	}

	for _, cr := range crews {
		name := cr.name
		if name == "night_a" {
			name = "night"
		}
		rows, err := s.pool.Query(ctx, `
			SELECT sender_id, COUNT(*) as cnt
			FROM messages
			WHERE created_at >= $1 AND created_at < $2
			AND EXTRACT(HOUR FROM created_at) >= $3
			AND EXTRACT(HOUR FROM created_at) < $4
			AND (expires_at IS NULL OR expires_at > now())
			GROUP BY sender_id
			ORDER BY cnt DESC
			LIMIT 5
		`, dayStart, dayEnd, cr.hourStart, cr.hourEnd)
		if err != nil {
			continue
		}
		var members []CrewMember
		for rows.Next() {
			var m CrewMember
			if err := rows.Scan(&m.SenderID, &m.Count); err != nil {
				continue
			}
			members = append(members, m)
		}
		rows.Close()
		if len(members) > 0 {
			snap.Crews[name] = append(snap.Crews[name], members...)
		}
	}

	// Night crew part 2: 0-6
	rows, err := s.pool.Query(ctx, `
		SELECT sender_id, COUNT(*) as cnt
		FROM messages
		WHERE created_at >= $1 AND created_at < $2
		AND EXTRACT(HOUR FROM created_at) < 6
		AND (expires_at IS NULL OR expires_at > now())
		GROUP BY sender_id
		ORDER BY cnt DESC
		LIMIT 5
	`, dayStart, dayEnd)
	if err == nil {
		for rows.Next() {
			var m CrewMember
			if err := rows.Scan(&m.SenderID, &m.Count); err != nil {
				continue
			}
			snap.Crews["night"] = append(snap.Crews["night"], m)
		}
		rows.Close()
	}

	// 4. Conversation pairs — consecutive messages within 5 min
	pairRows, err := s.pool.Query(ctx, `
		WITH ordered AS (
			SELECT sender_id,
				LAG(sender_id) OVER (ORDER BY created_at) AS prev_sender,
				created_at,
				LAG(created_at) OVER (ORDER BY created_at) AS prev_ts
			FROM messages
			WHERE created_at >= $1 AND created_at < $2
			AND (expires_at IS NULL OR expires_at > now())
		)
		SELECT
			LEAST(sender_id, prev_sender) AS a,
			GREATEST(sender_id, prev_sender) AS b,
			COUNT(*) AS cnt
		FROM ordered
		WHERE prev_sender IS NOT NULL
		AND sender_id != prev_sender
		AND created_at - prev_ts < INTERVAL '5 minutes'
		GROUP BY a, b
		ORDER BY cnt DESC
		LIMIT 5
	`, dayStart, dayEnd)
	if err == nil {
		for pairRows.Next() {
			var p ConversationPair
			if err := pairRows.Scan(&p.SenderA, &p.SenderB, &p.Count); err != nil {
				continue
			}
			snap.TopPairs = append(snap.TopPairs, p)
		}
		pairRows.Close()
	}

	// 5. Verb leader — count words from a verb list per sender
	verbRow := s.pool.QueryRow(ctx, `
		WITH words AS (
			SELECT sender_id, unnest(regexp_split_to_array(lower(content), '\s+')) AS word
			FROM messages
			WHERE created_at >= $1 AND created_at < $2
			AND content != ''
			AND (expires_at IS NULL OR expires_at > now())
		),
		verb_counts AS (
			SELECT sender_id, word, COUNT(*) AS cnt
			FROM words
			WHERE word IN (
				'said','think','went','believe','know','want','need','feel','see','make',
				'go','come','take','give','get','say','tell','ask','try','use',
				'find','put','call','run','keep','let','begin','seem','help','show',
				'hear','play','move','live','happen','bring','write','start','stop','read',
				'spend','grow','open','walk','win','teach','learn','lead','understand','watch',
				'follow','create','speak','buy','wait','serve','die','send','build','stay',
				'fall','cut','reach','kill','remain','suggest','raise','pass','sell','decide',
				'return','explain','hope','develop','carry','break','receive','agree','support','hold',
				'produce','eat','cover','catch','draw','choose','cause','point','listen','plan',
				'notice','enjoy','wonder','love','hate','remember','consider','appear','expect','wish'
			)
			GROUP BY sender_id, word
		),
		leader AS (
			SELECT sender_id, SUM(cnt)::int AS total,
				array_agg(word ORDER BY cnt DESC) AS top_words
			FROM verb_counts
			GROUP BY sender_id
			ORDER BY total DESC
			LIMIT 1
		)
		SELECT sender_id, total, top_words[1:5]
		FROM leader
	`, dayStart, dayEnd)

	var vl VerbLeader
	var samples []string
	if err := verbRow.Scan(&vl.SenderID, &vl.Count, &samples); err == nil {
		vl.Samples = samples
		snap.VerbLeader = &vl
	}

	// 6. Link of the day — prefer fetched links with titles
	linkRow := s.pool.QueryRow(ctx, `
		SELECT u.url, COALESCE(u.title, ''), m.sender_id
		FROM urls u
		JOIN messages m ON u.message_id = m.id
		WHERE m.created_at >= $1 AND m.created_at < $2
		AND (m.expires_at IS NULL OR m.expires_at > now())
		ORDER BY u.fetched DESC, u.created_at DESC
		LIMIT 1
	`, dayStart, dayEnd)

	var ld LinkOfDay
	if err := linkRow.Scan(&ld.URL, &ld.Title, &ld.SenderID); err == nil {
		snap.LinkOfDay = &ld
	}

	// 7. Yesterday ref — quote + link from yesterday's snapshot
	yesterday := dayStart.Add(-24 * time.Hour)
	var prevSnapshot json.RawMessage
	err = s.pool.QueryRow(ctx, `
		SELECT snapshot FROM daily_insights
		WHERE snapshot_date = $1
		AND snapshot IS NOT NULL AND snapshot != '{}'::jsonb
		ORDER BY created_at DESC LIMIT 1
	`, yesterday).Scan(&prevSnapshot)
	if err == nil && len(prevSnapshot) > 2 {
		var prevSnap DaySnapshot
		if json.Unmarshal(prevSnapshot, &prevSnap) == nil {
			ref := &YesterdayRef{}
			hasRef := false

			// Get yesterday's quote
			var prevQuote string
			_ = s.pool.QueryRow(ctx, `
				SELECT quote_content FROM daily_insights
				WHERE snapshot_date = $1 AND quote_content != ''
				ORDER BY created_at DESC LIMIT 1
			`, yesterday).Scan(&prevQuote)
			if prevQuote != "" {
				ref.Quote = prevQuote
				hasRef = true
			}

			if prevSnap.LinkOfDay != nil {
				ref.Link = prevSnap.LinkOfDay
				hasRef = true
			}

			if hasRef {
				snap.YesterdayRef = ref
			}
		}
	}

	return snap, nil
}

func (s *Store) ComputeWeeklyExtras(ctx context.Context, sundayDate time.Time) (weeklyTotal int, busiestDay string, busiestDayCount int) {
	loc := sundayDate.Location()
	// Monday of this week
	weekStart := time.Date(sundayDate.Year(), sundayDate.Month(), sundayDate.Day(), 0, 0, 0, 0, loc)
	weekStart = weekStart.Add(-6 * 24 * time.Hour) // Go back 6 days to Monday
	weekEnd := time.Date(sundayDate.Year(), sundayDate.Month(), sundayDate.Day(), 0, 0, 0, 0, loc).Add(24 * time.Hour)

	// Weekly total
	_ = s.pool.QueryRow(ctx, `
		SELECT COUNT(*) FROM messages
		WHERE created_at >= $1 AND created_at < $2
		AND (expires_at IS NULL OR expires_at > now())
	`, weekStart, weekEnd).Scan(&weeklyTotal)

	// Busiest day
	var busiestDate time.Time
	_ = s.pool.QueryRow(ctx, `
		SELECT date_trunc('day', created_at)::date AS d, COUNT(*) AS cnt
		FROM messages
		WHERE created_at >= $1 AND created_at < $2
		AND (expires_at IS NULL OR expires_at > now())
		GROUP BY d
		ORDER BY cnt DESC
		LIMIT 1
	`, weekStart, weekEnd).Scan(&busiestDate, &busiestDayCount)

	if !busiestDate.IsZero() {
		busiestDay = busiestDate.Format("Monday")
	}

	return
}

func (s *Store) GetDailySnapshots(ctx context.Context, days int) ([]DailyInsight, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, overview, themes, COALESCE(quote_content,''), COALESCE(quote_sender,''),
			quote_created_at, COALESCE(image_path,''), COALESCE(superlatives, '[]'::jsonb),
			COALESCE(snapshot, '{}'::jsonb), snapshot_date, created_at
		FROM daily_insights
		WHERE snapshot_date IS NOT NULL
		ORDER BY snapshot_date DESC
		LIMIT $1
	`, days)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []DailyInsight
	for rows.Next() {
		var di DailyInsight
		if err := rows.Scan(
			&di.ID, &di.Overview, &di.Themes, &di.QuoteContent, &di.QuoteSender,
			&di.QuoteCreatedAt, &di.ImagePath, &di.Superlatives,
			&di.Snapshot, &di.SnapshotDate, &di.CreatedAt,
		); err != nil {
			return nil, err
		}
		results = append(results, di)
	}
	return results, nil
}
