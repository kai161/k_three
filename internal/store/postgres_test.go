package store

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresActivationAndIdempotentCompletion(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := New(pool)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE feedback, entries, activations, devices"); err != nil {
		t.Fatal(err)
	}

	if err := store.EnsureDevice(ctx, "device-one"); err != nil {
		t.Fatal(err)
	}
	if err := store.RegisterActivation(ctx, "TEST-CODE"); err != nil {
		t.Fatal(err)
	}
	if err := store.Activate(ctx, "device-one", "TEST-CODE"); err != nil {
		t.Fatal(err)
	}
	progress, err := store.Progress(ctx, "device-one")
	if err != nil || !progress.Activated {
		t.Fatalf("activation failed: progress=%+v err=%v", progress, err)
	}

	if _, err := store.Complete(ctx, "device-one", 1, "平静", "揉掉"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Complete(ctx, "device-one", 1, "疲惫", "封存"); err != nil {
		t.Fatal(err)
	}
	progress, err = store.Progress(ctx, "device-one")
	if err != nil {
		t.Fatal(err)
	}
	if len(progress.Entries) != 1 || progress.Entries[0].Mood != "平静" {
		t.Fatalf("completion was not idempotent: %+v", progress.Entries)
	}

	var rawReflection string
	err = pool.QueryRow(ctx, "SELECT COALESCE(MAX(mood), '') FROM entries WHERE mood = '今天开了很多会'").Scan(&rawReflection)
	if err != nil {
		t.Fatal(err)
	}
	if rawReflection != "" {
		t.Fatal("raw reflection must not be stored")
	}
}

func TestPostgresFeedbackUpsert(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := New(pool)
	if err := store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "TRUNCATE feedback, entries, activations, devices"); err != nil {
		t.Fatal(err)
	}
	if err := store.EnsureDevice(ctx, "feedback-device"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFeedback(ctx, "feedback-device", 2, "第一次反馈"); err != nil {
		t.Fatal(err)
	}
	if err := store.SaveFeedback(ctx, "feedback-device", 2, "更新后的反馈"); err != nil {
		t.Fatal(err)
	}
	var count int
	var text string
	if err := pool.QueryRow(ctx, "SELECT COUNT(*), MAX(text) FROM feedback WHERE device_id = 'feedback-device' AND day = 2").Scan(&count, &text); err != nil {
		t.Fatal(err)
	}
	if count != 1 || text != "更新后的反馈" {
		t.Fatalf("feedback was not updated: count=%d text=%q", count, text)
	}
}
