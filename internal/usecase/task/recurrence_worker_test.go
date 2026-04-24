package task

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCalculateNextRunDate_DailyDefault(t *testing.T) {
	current := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)

	next, err := calculateNextRunDate("daily", json.RawMessage(`{}`), current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next == nil {
		t.Fatal("expected next run date, got nil")
	}

	expected := current.AddDate(0, 0, 1)
	if !next.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, *next)
	}
}

func TestCalculateNextRunDate_DatesNoFuture(t *testing.T) {
	current := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	cfg := json.RawMessage(`{"dates":["2026-04-20","2026-04-24"]}`)

	next, err := calculateNextRunDate("dates", cfg, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next != nil {
		t.Fatalf("expected nil next run date, got %v", *next)
	}
}

func TestCalculateNextRunDate_EvenOdd(t *testing.T) {
	current := time.Date(2026, 4, 24, 10, 0, 0, 0, time.UTC)
	cfg := json.RawMessage(`{"mode":"odd"}`)

	next, err := calculateNextRunDate("even_odd", cfg, current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next == nil {
		t.Fatal("expected next run date, got nil")
	}

	expected := time.Date(2026, 4, 25, 10, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, *next)
	}
}
