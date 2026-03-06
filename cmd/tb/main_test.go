package main

import (
	"testing"
	"time"
)

func TestToSlug(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"ACME Corp", "acme-corp"},
		{"TechStart Inc.", "techstart-inc"},
		{"hello world", "hello-world"},
		{"  spaces  ", "spaces"},
		{"already-slug", "already-slug"},
		{"foo--bar", "foo-bar"},
		{"123 Numbers", "123-numbers"},
		{"trailing-", "trailing"},
	}
	for _, tc := range cases {
		if got := toSlug(tc.input); got != tc.want {
			t.Errorf("toSlug(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestParseAgo(t *testing.T) {
	cases := []struct {
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"2h ago", 2 * time.Hour, false},
		{"30m ago", 30 * time.Minute, false},
		{"1h30m ago", 90 * time.Minute, false},
		{"45m", 45 * time.Minute, false},
		{"2h30mago", 150 * time.Minute, false},
		{"bad", 0, true},
		{"xyz ago", 0, true},
	}
	for _, tc := range cases {
		got, err := parseAgo(tc.input)
		if tc.wantErr {
			if err == nil {
				t.Errorf("parseAgo(%q): expected error, got nil", tc.input)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseAgo(%q): unexpected error: %v", tc.input, err)
			continue
		}
		if got != tc.want {
			t.Errorf("parseAgo(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}
}

func TestParsePeriod_Empty(t *testing.T) {
	now := time.Now()
	start, end, err := parsePeriod("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start.Day() != 1 {
		t.Errorf("expected day 1, got %d", start.Day())
	}
	if start.Month() != now.Month() || start.Year() != now.Year() {
		t.Errorf("expected current month %v, got %v", now.Month(), start.Month())
	}
	if end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Errorf("expected end of day 23:59:59, got %v", end)
	}
}

func TestParsePeriod_Specific(t *testing.T) {
	start, end, err := parsePeriod("2025-02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start.Year() != 2025 || start.Month() != 2 || start.Day() != 1 {
		t.Errorf("unexpected start: %v", start)
	}
	// 2025 is not a leap year; February ends on day 28.
	if end.Year() != 2025 || end.Month() != 2 || end.Day() != 28 {
		t.Errorf("unexpected end: %v", end)
	}
}

func TestParsePeriod_LeapYear(t *testing.T) {
	start, end, err := parsePeriod("2024-02")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if start.Day() != 1 {
		t.Errorf("expected day 1, got %d", start.Day())
	}
	// 2024 is a leap year; February ends on day 29.
	if end.Day() != 29 {
		t.Errorf("expected day 29, got %d", end.Day())
	}
}

func TestParsePeriod_Invalid(t *testing.T) {
	_, _, err := parsePeriod("not-a-period")
	if err == nil {
		t.Error("expected error for invalid period")
	}
}
