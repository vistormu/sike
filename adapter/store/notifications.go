package store

import (
	"context"
	"time"
)

// WasSent: the notifier already sent key to the person.
func (s *Store) WasSent(ctx context.Context, userID int64, key string) (bool, error) {
	var n int
	err := s.db.GetContext(ctx, &n, `SELECT count(*) FROM notifications_sent WHERE user_id = ? AND key = ?`, userID, key)
	return n > 0, err
}

// MarkSent writes down that key went to the person at at.
func (s *Store) MarkSent(ctx context.Context, userID int64, key string, at time.Time) error {
	_, err := s.db.ExecContext(ctx, `INSERT OR IGNORE INTO notifications_sent (user_id, key, sent_at) VALUES (?, ?, ?)`,
		userID, key, at.UTC().Format(time.RFC3339))
	return err
}

// ForgetSent drops what was sent before before.
func (s *Store) ForgetSent(ctx context.Context, before time.Time) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM notifications_sent WHERE sent_at < ?`, before.UTC().Format(time.RFC3339))
	return err
}
