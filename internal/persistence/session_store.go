package persistence

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"omenpath-lab/internal/config"
)

// CreateSession atomically publishes a complete canonical laboratory and its
// token hash. It deliberately accepts no raw cookie material.
func (s *Store) CreateSession(ctx context.Context, id LabID, hash [32]byte, now time.Time, cfg config.Config) error {
	repo, err := s.ForLab(id)
	if err != nil {
		return err
	}
	if now.IsZero() || now.Location() != time.UTC || cfg.SessionTTL <= 0 {
		return fmt.Errorf("create session: invalid input")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin session creation: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	if err := repo.bootstrapTx(ctx, tx, now, cfg); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO sessions(token_hash,lab_id,expires_at,last_seen_at) VALUES(?,?,?,?)`, hash[:], id, now.Add(cfg.SessionTTL).UnixNano(), now.UnixNano()); err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return tx.Commit()
}

// ResolveSession returns an empty LabID for an unknown or expired hash.
// Refreshed tells the transport to reissue its incoming cookie; raw tokens never
// cross this persistence boundary. The single Store connection serializes this
// transaction with cleanup, including the read-before-refresh interval.
func (s *Store) ResolveSession(ctx context.Context, hash [32]byte, now time.Time, cfg config.Config) (id LabID, expires time.Time, refreshed bool, err error) {
	if s == nil || s.db == nil || now.IsZero() || cfg.SessionTTL <= 0 || cfg.SessionRefreshInterval <= 0 {
		return "", time.Time{}, false, fmt.Errorf("resolve session: invalid input")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return "", time.Time{}, false, err
	}
	defer func() { _ = tx.Rollback() }()
	var expiry, seen int64
	err = tx.QueryRowContext(ctx, `SELECT sessions.lab_id,sessions.expires_at,sessions.last_seen_at FROM sessions JOIN labs ON labs.id=sessions.lab_id WHERE token_hash=? AND sessions.expires_at>? AND labs.expires_at>?`, hash[:], now.UnixNano(), now.UnixNano()).Scan(&id, &expiry, &seen)
	if errors.Is(err, sql.ErrNoRows) {
		return "", time.Time{}, false, nil
	}
	if err != nil {
		return "", time.Time{}, false, err
	}
	expires = time.Unix(0, expiry).UTC()
	if now.Sub(time.Unix(0, seen)) >= cfg.SessionRefreshInterval {
		expires = now.Add(cfg.SessionTTL)
		if _, err = tx.ExecContext(ctx, `UPDATE sessions SET expires_at=?,last_seen_at=? WHERE token_hash=? AND lab_id=?`, expires.UnixNano(), now.UnixNano(), hash[:], id); err != nil {
			return "", time.Time{}, false, err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE labs SET expires_at=?,last_active_at=? WHERE id=?`, expires.UnixNano(), now.UnixNano(), id); err != nil {
			return "", time.Time{}, false, err
		}
		refreshed = true
	}
	if err = tx.Commit(); err != nil {
		return "", time.Time{}, false, err
	}
	return id, expires, refreshed, nil
}

// ExpiredLabs lists cleanup candidates only. Callers must hold the runtime idle
// gate while invoking DeleteExpiredLab; a candidate is not deletion authority.
func (s *Store) ExpiredLabs(ctx context.Context, now time.Time) ([]LabID, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id FROM labs WHERE expires_at<=? ORDER BY id`, now.UnixNano())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []LabID{}
	for rows.Next() {
		var id LabID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// DeleteExpiredLab rechecks expiry in the deleting statement. A renewal that
// committed after candidate selection protects the lab and all of its rows.
func (s *Store) DeleteExpiredLab(ctx context.Context, id LabID, now time.Time) (bool, error) {
	if err := id.Validate(); err != nil {
		return false, err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM labs WHERE id=? AND expires_at<=?`, id, now.UnixNano())
	if err != nil {
		return false, err
	}
	n, err := result.RowsAffected()
	return n != 0, err
}
