package store

import (
	"context"
	"fmt"
)

func (s *Store) SaveAttachment(ctx context.Context, a AttachmentRecord) (string, error) {
	query := `
		INSERT INTO attachments (message_id, signal_attachment_id, content_type, filename, size)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`
	var id string
	err := s.pool.QueryRow(ctx, query,
		a.MessageID, a.SignalAttachmentID, a.ContentType, a.Filename, a.Size,
	).Scan(&id)
	return id, err
}

func (s *Store) GetAttachment(ctx context.Context, id string) (*AttachmentRecord, error) {
	query := `
		SELECT id, message_id, signal_attachment_id, content_type, COALESCE(filename,''), size,
			COALESCE(local_path,''), downloaded, created_at
		FROM attachments WHERE id = $1
	`
	var a AttachmentRecord
	err := s.pool.QueryRow(ctx, query, id).Scan(
		&a.ID, &a.MessageID, &a.SignalAttachmentID, &a.ContentType, &a.Filename, &a.Size,
		&a.LocalPath, &a.Downloaded, &a.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (s *Store) ListAttachmentsByMessage(ctx context.Context, messageID string) ([]AttachmentRecord, error) {
	query := `
		SELECT id, message_id, signal_attachment_id, content_type, COALESCE(filename,''), size,
			COALESCE(local_path,''), downloaded, created_at
		FROM attachments WHERE message_id = $1
		ORDER BY created_at ASC
	`
	rows, err := s.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []AttachmentRecord
	for rows.Next() {
		var a AttachmentRecord
		if err := rows.Scan(
			&a.ID, &a.MessageID, &a.SignalAttachmentID, &a.ContentType, &a.Filename, &a.Size,
			&a.LocalPath, &a.Downloaded, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, nil
}

func (s *Store) ListAllAttachments(ctx context.Context, limit, offset int, sortBy ...string) ([]AttachmentRecord, int, error) {
	if limit <= 0 {
		limit = 50
	}

	var total int
	if err := s.pool.QueryRow(ctx, "SELECT COUNT(*) FROM attachments").Scan(&total); err != nil {
		return nil, 0, err
	}

	orderClause := "created_at DESC"
	if len(sortBy) > 0 {
		switch sortBy[0] {
		case "date_asc":
			orderClause = "created_at ASC"
		case "size_desc":
			orderClause = "size DESC"
		case "size_asc":
			orderClause = "size ASC"
		case "type":
			orderClause = "content_type ASC, created_at DESC"
		default: // date_desc
			orderClause = "created_at DESC"
		}
	}

	query := fmt.Sprintf(`
		SELECT id, message_id, signal_attachment_id, content_type, COALESCE(filename,''), size,
			COALESCE(local_path,''), downloaded, created_at
		FROM attachments
		ORDER BY %s
		LIMIT $1 OFFSET $2
	`, orderClause)
	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var attachments []AttachmentRecord
	for rows.Next() {
		var a AttachmentRecord
		if err := rows.Scan(
			&a.ID, &a.MessageID, &a.SignalAttachmentID, &a.ContentType, &a.Filename, &a.Size,
			&a.LocalPath, &a.Downloaded, &a.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		attachments = append(attachments, a)
	}
	return attachments, total, nil
}

func (s *Store) MarkAttachmentDownloaded(ctx context.Context, id, localPath string) error {
	query := `UPDATE attachments SET downloaded = true, local_path = $2 WHERE id = $1`
	_, err := s.pool.Exec(ctx, query, id, localPath)
	return err
}

func (s *Store) GetUndownloadedAttachments(ctx context.Context) ([]AttachmentRecord, error) {
	query := `
		SELECT id, message_id, signal_attachment_id, content_type, COALESCE(filename,''), size,
			COALESCE(local_path,''), downloaded, created_at
		FROM attachments WHERE downloaded = false
		ORDER BY created_at ASC
		LIMIT 100
	`
	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var attachments []AttachmentRecord
	for rows.Next() {
		var a AttachmentRecord
		if err := rows.Scan(
			&a.ID, &a.MessageID, &a.SignalAttachmentID, &a.ContentType, &a.Filename, &a.Size,
			&a.LocalPath, &a.Downloaded, &a.CreatedAt,
		); err != nil {
			return nil, err
		}
		attachments = append(attachments, a)
	}
	return attachments, nil
}
