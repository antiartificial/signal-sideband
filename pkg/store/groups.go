package store

import (
	"context"
)

func (s *Store) UpsertGroup(ctx context.Context, g GroupRecord) error {
	query := `
		INSERT INTO groups (group_id, name, description, avatar_path, member_count)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (group_id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			avatar_path = EXCLUDED.avatar_path,
			member_count = EXCLUDED.member_count,
			updated_at = now()
	`
	_, err := s.pool.Exec(ctx, query, g.GroupID, g.Name, g.Description, g.AvatarPath, g.MemberCount)
	return err
}

type GroupWithCount struct {
	GroupRecord
	MessageCount int `json:"message_count"`
}

func (s *Store) ListGroups(ctx context.Context) ([]GroupWithCount, error) {
	query := `
		SELECT g.id, g.group_id, g.name, g.description, g.avatar_path, g.member_count,
			g.created_at, g.updated_at,
			COUNT(m.id) AS message_count
		FROM groups g
		LEFT JOIN messages m ON m.group_id = g.group_id
		GROUP BY g.id
		ORDER BY g.name ASC
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []GroupWithCount
	for rows.Next() {
		var g GroupWithCount
		if err := rows.Scan(
			&g.ID, &g.GroupID, &g.Name, &g.Description, &g.AvatarPath, &g.MemberCount,
			&g.CreatedAt, &g.UpdatedAt, &g.MessageCount,
		); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, nil
}

func (s *Store) GetGroupByID(ctx context.Context, groupID string) (*GroupRecord, error) {
	query := `
		SELECT id, group_id, name, description, avatar_path, member_count, created_at, updated_at
		FROM groups WHERE group_id = $1
	`
	var g GroupRecord
	err := s.pool.QueryRow(ctx, query, groupID).Scan(
		&g.ID, &g.GroupID, &g.Name, &g.Description, &g.AvatarPath, &g.MemberCount,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &g, nil
}
