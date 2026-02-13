package store

import (
	"context"
	"strconv"
	"time"
)

func (s *Store) UpsertConcept(ctx context.Context, c CerebroConcept) (string, error) {
	query := `
		INSERT INTO cerebro_concepts (name, category, description, metadata, group_id, first_seen, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $6)
		ON CONFLICT (lower(name), coalesce(group_id, '__global__'))
		DO UPDATE SET
			mention_count = cerebro_concepts.mention_count + 1,
			last_seen = EXCLUDED.last_seen,
			description = CASE WHEN EXCLUDED.description != '' THEN EXCLUDED.description ELSE cerebro_concepts.description END
		RETURNING id
	`
	metadata := c.Metadata
	if metadata == nil {
		metadata = []byte("{}")
	}
	lastSeen := c.LastSeen
	if lastSeen.IsZero() {
		lastSeen = time.Now()
	}
	var id string
	err := s.pool.QueryRow(ctx, query,
		c.Name, c.Category, c.Description, metadata, c.GroupID, lastSeen,
	).Scan(&id)
	return id, err
}

func (s *Store) UpsertEdge(ctx context.Context, e CerebroEdge) (string, error) {
	query := `
		INSERT INTO cerebro_edges (source_id, target_id, relation)
		VALUES ($1, $2, $3)
		ON CONFLICT (source_id, target_id, relation)
		DO UPDATE SET weight = cerebro_edges.weight + 1
		RETURNING id
	`
	var id string
	err := s.pool.QueryRow(ctx, query, e.SourceID, e.TargetID, e.Relation).Scan(&id)
	return id, err
}

func (s *Store) SaveEnrichment(ctx context.Context, e CerebroEnrichment) (string, error) {
	query := `
		INSERT INTO cerebro_enrichments (concept_id, source, content, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`
	var id string
	err := s.pool.QueryRow(ctx, query, e.ConceptID, e.Source, e.Content, e.ExpiresAt).Scan(&id)
	return id, err
}

func (s *Store) SaveExtraction(ctx context.Context, e CerebroExtraction) (string, error) {
	query := `
		INSERT INTO cerebro_extractions (batch_start, batch_end, message_count, concept_count, edge_count, llm_provider, llm_model, token_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
	`
	var id string
	err := s.pool.QueryRow(ctx, query,
		e.BatchStart, e.BatchEnd, e.MessageCount, e.ConceptCount, e.EdgeCount,
		e.LLMProvider, e.LLMModel, e.TokenCount,
	).Scan(&id)
	return id, err
}

func (s *Store) GetLastExtractionTime(ctx context.Context) (*time.Time, error) {
	query := `SELECT MAX(batch_end) FROM cerebro_extractions`
	var t *time.Time
	err := s.pool.QueryRow(ctx, query).Scan(&t)
	if err != nil {
		return nil, err
	}
	return t, nil
}

func (s *Store) GetCerebroGraph(ctx context.Context, groupID *string, since *time.Time, limit int) (*CerebroGraph, error) {
	if limit <= 0 {
		limit = 50
	}

	// Get top concepts
	var conceptQuery string
	var args []any
	argIdx := 1

	conceptQuery = `
		WITH top_concepts AS (
			SELECT id, name, category, description, mention_count, first_seen, last_seen, metadata, group_id, created_at
			FROM cerebro_concepts
			WHERE 1=1
	`
	if groupID != nil {
		conceptQuery += ` AND group_id = $` + itoa(argIdx)
		args = append(args, *groupID)
		argIdx++
	}
	if since != nil {
		conceptQuery += ` AND last_seen >= $` + itoa(argIdx)
		args = append(args, *since)
		argIdx++
	}
	conceptQuery += `
			ORDER BY mention_count DESC
			LIMIT $` + itoa(argIdx) + `
		)
		SELECT id, name, category, description, mention_count, first_seen, last_seen, metadata, group_id, created_at
		FROM top_concepts
	`
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, conceptQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concepts []CerebroConcept
	var conceptIDs []string
	for rows.Next() {
		var c CerebroConcept
		if err := rows.Scan(&c.ID, &c.Name, &c.Category, &c.Description, &c.MentionCount,
			&c.FirstSeen, &c.LastSeen, &c.Metadata, &c.GroupID, &c.CreatedAt); err != nil {
			return nil, err
		}
		concepts = append(concepts, c)
		conceptIDs = append(conceptIDs, c.ID)
	}

	if len(concepts) == 0 {
		return &CerebroGraph{Concepts: []CerebroConcept{}, Edges: []CerebroEdge{}}, nil
	}

	// Get edges between these concepts
	edgeQuery := `
		SELECT id, source_id, target_id, relation, weight
		FROM cerebro_edges
		WHERE source_id = ANY($1) AND target_id = ANY($1)
	`
	edgeRows, err := s.pool.Query(ctx, edgeQuery, conceptIDs)
	if err != nil {
		return nil, err
	}
	defer edgeRows.Close()

	var edges []CerebroEdge
	for edgeRows.Next() {
		var e CerebroEdge
		if err := edgeRows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.Relation, &e.Weight); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	if edges == nil {
		edges = []CerebroEdge{}
	}

	return &CerebroGraph{Concepts: concepts, Edges: edges}, nil
}

func (s *Store) GetConceptDetail(ctx context.Context, id string) (*CerebroConceptDetail, error) {
	// Get concept
	cQuery := `
		SELECT id, name, category, description, mention_count, first_seen, last_seen, metadata, group_id, created_at
		FROM cerebro_concepts WHERE id = $1
	`
	var c CerebroConcept
	err := s.pool.QueryRow(ctx, cQuery, id).Scan(
		&c.ID, &c.Name, &c.Category, &c.Description, &c.MentionCount,
		&c.FirstSeen, &c.LastSeen, &c.Metadata, &c.GroupID, &c.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Get edges
	eQuery := `
		SELECT id, source_id, target_id, relation, weight
		FROM cerebro_edges
		WHERE source_id = $1 OR target_id = $1
	`
	eRows, err := s.pool.Query(ctx, eQuery, id)
	if err != nil {
		return nil, err
	}
	defer eRows.Close()

	var edges []CerebroEdge
	for eRows.Next() {
		var e CerebroEdge
		if err := eRows.Scan(&e.ID, &e.SourceID, &e.TargetID, &e.Relation, &e.Weight); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	if edges == nil {
		edges = []CerebroEdge{}
	}

	// Get non-expired enrichments
	enrichments, err := s.GetConceptEnrichments(ctx, id)
	if err != nil {
		return nil, err
	}

	return &CerebroConceptDetail{
		CerebroConcept: c,
		Edges:          edges,
		Enrichments:    enrichments,
	}, nil
}

func (s *Store) GetConceptEnrichments(ctx context.Context, conceptID string) ([]CerebroEnrichment, error) {
	query := `
		SELECT id, concept_id, source, content, expires_at, created_at
		FROM cerebro_enrichments
		WHERE concept_id = $1 AND (expires_at IS NULL OR expires_at > now())
		ORDER BY created_at DESC
	`
	rows, err := s.pool.Query(ctx, query, conceptID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var enrichments []CerebroEnrichment
	for rows.Next() {
		var e CerebroEnrichment
		if err := rows.Scan(&e.ID, &e.ConceptID, &e.Source, &e.Content, &e.ExpiresAt, &e.CreatedAt); err != nil {
			return nil, err
		}
		enrichments = append(enrichments, e)
	}
	if enrichments == nil {
		enrichments = []CerebroEnrichment{}
	}
	return enrichments, nil
}

func (s *Store) DeleteExpiredEnrichments(ctx context.Context) (int, error) {
	result, err := s.pool.Exec(ctx, `DELETE FROM cerebro_enrichments WHERE expires_at IS NOT NULL AND expires_at <= now()`)
	if err != nil {
		return 0, err
	}
	return int(result.RowsAffected()), nil
}

func (s *Store) GetConceptsNeedingEnrichment(ctx context.Context, limit int) ([]CerebroConcept, error) {
	query := `
		SELECT c.id, c.name, c.category, c.description, c.mention_count, c.first_seen, c.last_seen, c.metadata, c.group_id, c.created_at
		FROM cerebro_concepts c
		LEFT JOIN cerebro_enrichments e ON e.concept_id = c.id AND (e.expires_at IS NULL OR e.expires_at > now())
		WHERE e.id IS NULL
		ORDER BY c.mention_count DESC
		LIMIT $1
	`
	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var concepts []CerebroConcept
	for rows.Next() {
		var c CerebroConcept
		if err := rows.Scan(&c.ID, &c.Name, &c.Category, &c.Description, &c.MentionCount,
			&c.FirstSeen, &c.LastSeen, &c.Metadata, &c.GroupID, &c.CreatedAt); err != nil {
			return nil, err
		}
		concepts = append(concepts, c)
	}
	return concepts, nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
