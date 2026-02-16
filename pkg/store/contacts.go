package store

import (
	"context"
)

func (s *Store) UpsertContact(ctx context.Context, c ContactRecord) error {
	query := `
		INSERT INTO contacts (source_uuid, phone_number, profile_name, alias, avatar_path)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (source_uuid) DO UPDATE SET
			phone_number = CASE WHEN EXCLUDED.phone_number != '' THEN EXCLUDED.phone_number ELSE contacts.phone_number END,
			profile_name = CASE WHEN EXCLUDED.profile_name != '' THEN EXCLUDED.profile_name ELSE contacts.profile_name END,
			avatar_path = CASE WHEN EXCLUDED.avatar_path != '' THEN EXCLUDED.avatar_path ELSE contacts.avatar_path END,
			updated_at = now()
	`
	_, err := s.pool.Exec(ctx, query, c.SourceUUID, c.PhoneNumber, c.ProfileName, c.Alias, c.AvatarPath)
	return err
}

func (s *Store) GetContactByUUID(ctx context.Context, uuid string) (*ContactRecord, error) {
	query := `
		SELECT id, source_uuid, COALESCE(phone_number, ''), COALESCE(profile_name, ''), COALESCE(alias, ''), COALESCE(avatar_path, ''), created_at, updated_at
		FROM contacts WHERE source_uuid = $1
	`
	var c ContactRecord
	err := s.pool.QueryRow(ctx, query, uuid).Scan(
		&c.ID, &c.SourceUUID, &c.PhoneNumber, &c.ProfileName, &c.Alias, &c.AvatarPath,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) ListContacts(ctx context.Context) ([]ContactRecord, error) {
	query := `
		SELECT id, source_uuid, COALESCE(phone_number, ''), COALESCE(profile_name, ''), COALESCE(alias, ''), COALESCE(avatar_path, ''), created_at, updated_at
		FROM contacts ORDER BY COALESCE(NULLIF(alias, ''), profile_name) ASC
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contacts []ContactRecord
	for rows.Next() {
		var c ContactRecord
		if err := rows.Scan(
			&c.ID, &c.SourceUUID, &c.PhoneNumber, &c.ProfileName, &c.Alias, &c.AvatarPath,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}

func (s *Store) UpdateContactAlias(ctx context.Context, uuid, alias string) error {
	query := `
		INSERT INTO contacts (source_uuid, alias)
		VALUES ($1, $2)
		ON CONFLICT (source_uuid) DO UPDATE SET
			alias = EXCLUDED.alias,
			updated_at = now()
	`
	_, err := s.pool.Exec(ctx, query, uuid, alias)
	return err
}

func (s *Store) ListDistinctSenders(ctx context.Context) ([]DistinctSender, error) {
	query := `
		SELECT COALESCE(MIN(sender_id), ''), COALESCE(MIN(source_uuid), MIN(sender_id), ''), COUNT(*) as message_count
		FROM messages
		WHERE NOT is_outgoing AND (source_uuid IS NOT NULL OR sender_id != '')
		GROUP BY COALESCE(source_uuid, sender_id)
		ORDER BY message_count DESC
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var senders []DistinctSender
	for rows.Next() {
		var d DistinctSender
		if err := rows.Scan(&d.SenderID, &d.SourceUUID, &d.MessageCount); err != nil {
			return nil, err
		}
		senders = append(senders, d)
	}
	return senders, nil
}
