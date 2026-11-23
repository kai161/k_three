package store

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"unwind/internal/plan"
	"unwind/migrations"
)

var ErrInvalidActivation = errors.New("invalid or already used activation code")

// Postgres stores only anonymous progress and entitlement data.
type Postgres struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Postgres {
	return &Postgres{pool: pool}
}

func (s *Postgres) Migrate(ctx context.Context) error {
	if _, err := s.pool.Exec(ctx, migrations.SQL); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, migrations.FeedbackSQL)
	return err
}

func (s *Postgres) EnsureDevice(ctx context.Context, deviceID string) error {
	_, err := s.pool.Exec(ctx, "INSERT INTO devices (id) VALUES ($1) ON CONFLICT (id) DO NOTHING", deviceID)
	return err
}

func (s *Postgres) Progress(ctx context.Context, deviceID string) (plan.Progress, error) {
	progress := plan.Progress{DeviceID: deviceID, Entries: []plan.Entry{}}
	if err := s.pool.QueryRow(ctx, "SELECT activated FROM devices WHERE id = $1", deviceID).Scan(&progress.Activated); err != nil {
		return plan.Progress{}, err
	}
	rows, err := s.pool.Query(ctx, "SELECT day, mood, ritual, completed_at FROM entries WHERE device_id = $1 ORDER BY day", deviceID)
	if err != nil {
		return plan.Progress{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var entry plan.Entry
		if err := rows.Scan(&entry.Day, &entry.Mood, &entry.Ritual, &entry.CompletedAt); err != nil {
			return plan.Progress{}, err
		}
		progress.Entries = append(progress.Entries, entry)
	}
	return progress, rows.Err()
}

func (s *Postgres) Complete(ctx context.Context, deviceID string, day int, mood, ritual string) (plan.Entry, error) {
	entry := plan.Entry{}
	err := s.pool.QueryRow(ctx, `
		INSERT INTO entries (device_id, day, mood, ritual)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (device_id, day) DO NOTHING
		RETURNING day, mood, ritual, completed_at`, deviceID, day, mood, ritual).
		Scan(&entry.Day, &entry.Mood, &entry.Ritual, &entry.CompletedAt)
	if err == nil {
		return entry, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return plan.Entry{}, err
	}
	err = s.pool.QueryRow(ctx, "SELECT day, mood, ritual, completed_at FROM entries WHERE device_id = $1 AND day = $2", deviceID, day).
		Scan(&entry.Day, &entry.Mood, &entry.Ritual, &entry.CompletedAt)
	return entry, err
}

func (s *Postgres) RegisterActivation(ctx context.Context, code string) error {
	_, err := s.pool.Exec(ctx, "INSERT INTO activations (code_hash) VALUES ($1) ON CONFLICT (code_hash) DO NOTHING", hash(code))
	return err
}

func (s *Postgres) Activate(ctx context.Context, deviceID, code string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var boundDevice string
	err = tx.QueryRow(ctx, `
		UPDATE activations
		SET device_id = $2, activated_at = NOW()
		WHERE code_hash = $1 AND (device_id IS NULL OR device_id = $2)
		RETURNING device_id`, hash(code), deviceID).Scan(&boundDevice)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrInvalidActivation
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "UPDATE devices SET activated = TRUE WHERE id = $1", deviceID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit activation: %w", err)
	}
	return nil
}

func (s *Postgres) SaveFeedback(ctx context.Context, deviceID string, day int, text string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO feedback (device_id, day, text)
		VALUES ($1, $2, $3)
		ON CONFLICT (device_id, day) DO UPDATE
		SET text = EXCLUDED.text, updated_at = NOW()`, deviceID, day, text)
	return err
}

func hash(value string) string {
	digest := sha256.Sum256([]byte(value))
	return hex.EncodeToString(digest[:])
}
