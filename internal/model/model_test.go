package model

import (
	"math"
	"testing"
	"time"
)

func TestSession_Duration_Running(t *testing.T) {
	before := time.Now()
	s := Session{
		ID:    "s1",
		Start: before.Add(-2 * time.Hour),
	}

	d := s.Duration()

	if d < 2*time.Hour {
		t.Errorf("expected duration >= 2h, got %v", d)
	}

	// End is zero, so Duration uses time.Now(); allow 1s of tolerance.
	if d > 2*time.Hour+time.Second {
		t.Errorf("duration too large for a 2h running session: %v", d)
	}
}

func TestSession_Duration_Finished(t *testing.T) {
	start := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 11, 30, 0, 0, time.UTC)
	s := Session{
		ID:    "s2",
		Start: start,
		End:   end,
	}

	d := s.Duration()

	if want := 2*time.Hour + 30*time.Minute; d != want {
		t.Errorf("expected %v, got %v", want, d)
	}
}

func TestSession_Hours(t *testing.T) {
	start := time.Date(2026, 3, 1, 8, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 1, 9, 30, 0, 0, time.UTC)
	s := Session{
		ID:    "s3",
		Start: start,
		End:   end,
	}

	h := s.Hours()

	if want := 1.5; math.Abs(h-want) > 1e-9 {
		t.Errorf("expected %.4f, got %.4f", want, h)
	}
}

func TestSession_Hours_Running(t *testing.T) {
	s := Session{
		ID:    "s4",
		Start: time.Now().Add(-30 * time.Minute),
	}

	h := s.Hours()

	if h < 0.5 {
		t.Errorf("expected hours >= 0.5 for a 30-min running session, got %.4f", h)
	}
}
