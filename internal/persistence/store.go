package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/url"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	seeddata "omenpath-lab/data"
	"omenpath-lab/internal/config"
	"omenpath-lab/internal/domain"
)

type Snapshot struct {
	Simulation domain.SimulationState
	App        domain.AppState
}

type Store struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		return nil, fmt.Errorf("open sqlite: empty path")
	}
	dsn, err := sqliteDSN(path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	store := &Store{db: db}
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return store, nil
}

func sqliteDSN(path string) (string, error) {
	settings := func(query url.Values) string {
		query.Del("_fk")
		query.Del("_timeout")
		query.Del("_pragma")
		query.Set("_busy_timeout", "5000")
		query.Set("_foreign_keys", "1")
		return query.Encode()
	}
	if path == ":memory:" {
		return path + "?" + settings(url.Values{}), nil
	}
	if parsed, err := url.Parse(path); err != nil {
		return "", fmt.Errorf("parse database URI: %w", err)
	} else if parsed.Scheme == "file" {
		parsed.RawQuery = settings(parsed.Query())
		return parsed.String(), nil
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve database path: %w", err)
	}
	return (&url.URL{Scheme: "file", Path: absolute, RawQuery: settings(url.Values{})}).String(), nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

type planeSeed struct {
	ID          int64    `json:"id"`
	Name        string   `json:"name"`
	Aliases     []string `json:"aliases"`
	CatalogTier string   `json:"catalog_tier"`
	Explored    bool     `json:"explored"`
}

type catalogSeed struct {
	Planes []planeSeed `json:"planes"`
}

func (s *LabRepository) Bootstrap(ctx context.Context, now time.Time, cfg config.Config) error {
	if !s.valid() || now.IsZero() || now.Location() != time.UTC {
		return fmt.Errorf("bootstrap: invalid input")
	}
	if cfg.ObserverCount != config.Default().ObserverCount || cfg.LabEnergyMax != 100 {
		return fmt.Errorf("bootstrap: noncanonical configuration")
	}

	tx, err := s.store.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bootstrap: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM app_state WHERE lab_id = ?`, s.labID).Scan(&exists); err != nil {
		return fmt.Errorf("inspect bootstrap state: %w", err)
	}
	if exists != 0 {
		return tx.Commit()
	}
	if err := s.bootstrapTx(ctx, tx, now, cfg); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bootstrap: %w", err)
	}
	return nil
}

// bootstrapTx is shared by explicit lab bootstrap and atomic session creation.
// The caller owns the transaction; no externally visible partial lab can exist.
func (s *LabRepository) bootstrapTx(ctx context.Context, tx *sql.Tx, now time.Time, cfg config.Config) error {
	var seed catalogSeed
	if err := json.Unmarshal(seeddata.MTGPlanesExpandedSeed, &seed); err != nil {
		return fmt.Errorf("decode plane seed: %w", err)
	}
	if len(seed.Planes) != 85 || cfg.ObserverCount != config.Default().ObserverCount || cfg.LabEnergyMax != 100 {
		return fmt.Errorf("bootstrap: noncanonical configuration")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO labs(id, created_at, expires_at, last_active_at)
		VALUES (?, ?, ?, ?)`, s.labID, now.UnixNano(), now.Add(cfg.SessionTTL).UnixNano(), now.UnixNano()); err != nil {
		return fmt.Errorf("insert laboratory: %w", err)
	}
	for _, plane := range seed.Planes {
		aliases, err := json.Marshal(plane.Aliases)
		if err != nil {
			return fmt.Errorf("encode aliases for plane %d: %w", plane.ID, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO planes
			(lab_id, id, name, aliases_json, catalog_tier, explored, explored_at)
			VALUES (?, ?, ?, ?, ?, 0, NULL)`,
			s.labID, plane.ID, plane.Name, string(aliases), plane.CatalogTier,
		); err != nil {
			return fmt.Errorf("insert plane %d: %w", plane.ID, err)
		}
	}
	for _, observer := range domain.NewObserverRoster(cfg.ObserverCount, now) {
		if _, err := tx.ExecContext(ctx, `INSERT INTO observers
			(lab_id, id, status, current_plane_id, active_portal_id, phase_started_at, phase_ends_at, created_at, updated_at)
			VALUES (?, ?, ?, NULL, NULL, NULL, NULL, ?, ?)`,
			s.labID, observer.ID, observer.Status, observer.CreatedAt.UnixNano(), observer.UpdatedAt.UnixNano(),
		); err != nil {
			return fmt.Errorf("insert observer %d: %w", observer.ID, err)
		}
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO lab_state
		(lab_id, energy_base, energy_base_at, override_until) VALUES (?, ?, ?, NULL)`,
		s.labID, cfg.LabEnergyMax, now.UnixNano(),
	); err != nil {
		return fmt.Errorf("insert lab state: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO app_state
		(lab_id, mode, tutorial_step, next_portal_id, spawn_scheduled_at, spawn_due_at, spawn_paused, last_tick_at)
		VALUES (?, ?, 0, 1, NULL, NULL, 1, NULL)`, s.labID, domain.ModeTutorial,
	); err != nil {
		return fmt.Errorf("insert app state: %w", err)
	}
	return nil
}
