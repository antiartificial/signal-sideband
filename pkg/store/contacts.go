package store

import (
	"context"
)

func (s *Store) UpsertContact(ctx context.Context, c ContactRecord) error {
	query := `
		INSERT INTO contacts (source_uuid, phone_number, profile_name, avatar_path)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (source_uuid) DO UPDATE SET
			phone_number = EXCLUDED.phone_number,
			profile_name = EXCLUDED.profile_name,
			avatar_path = EXCLUDED.avatar_path,
			updated_at = now()
	`
	_, err := s.pool.Exec(ctx, query, c.SourceUUID, c.PhoneNumber, c.ProfileName, c.AvatarPath)
	return err
}

func (s *Store) GetContactByUUID(ctx context.Context, uuid string) (*ContactRecord, error) {
	query := `
		SELECT id, source_uuid, phone_number, profile_name, avatar_path, created_at, updated_at
		FROM contacts WHERE source_uuid = $1
	`
	var c ContactRecord
	err := s.pool.QueryRow(ctx, query, uuid).Scan(
		&c.ID, &c.SourceUUID, &c.PhoneNumber, &c.ProfileName, &c.AvatarPath,
		&c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) ListContacts(ctx context.Context) ([]ContactRecord, error) {
	query := `
		SELECT id, source_uuid, phone_number, profile_name, avatar_path, created_at, updated_at
		FROM contacts ORDER BY profile_name ASC
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
			&c.ID, &c.SourceUUID, &c.PhoneNumber, &c.ProfileName, &c.AvatarPath,
			&c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contacts = append(contacts, c)
	}
	return contacts, nil
}
