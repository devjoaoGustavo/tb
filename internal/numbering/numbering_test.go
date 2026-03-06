package numbering

import (
	"fmt"
	"testing"
	"time"

	"github.com/joaogustavo/tb/internal/config"
	"github.com/joaogustavo/tb/internal/model"
)

func TestNextNumber_Default(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:          "INV",
			StartNumber:     1,
			Padding:         3,
			Separator:       "-",
			IncludeYear:     true,
			DefaultCurrency: model.CurrencyBRL,
		},
	}
	state := &config.State{NextInvoiceNumber: 1}
	year := time.Now().Year()

	num := NextNumber(cfg, state, "")
	if want := fmt.Sprintf("INV-%d-001", year); num != want {
		t.Errorf("expected %s, got %s", want, num)
	}
	if state.NextInvoiceNumber != 2 {
		t.Errorf("expected counter to advance to 2, got %d", state.NextInvoiceNumber)
	}

	num2 := NextNumber(cfg, state, "")
	if want := fmt.Sprintf("INV-%d-002", year); num2 != want {
		t.Errorf("expected %s, got %s", want, num2)
	}
	if state.NextInvoiceNumber != 3 {
		t.Errorf("expected counter to advance to 3, got %d", state.NextInvoiceNumber)
	}
}

func TestNextNumber_PerClient(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:          "INV",
			StartNumber:     1,
			Padding:         3,
			Separator:       "-",
			IncludeYear:     true,
			PerClientPrefix: true,
		},
	}
	state := &config.State{NextInvoiceNumber: 42}
	year := time.Now().Year()

	num := NextNumber(cfg, state, "acme")
	if want := fmt.Sprintf("ACME-%d-042", year); num != want {
		t.Errorf("expected %s, got %s", want, num)
	}
}

func TestNextNumber_NoYear(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:      "NF",
			StartNumber: 100,
			Padding:     4,
			Separator:   "-",
			IncludeYear: false,
		},
	}
	state := &config.State{NextInvoiceNumber: 100}

	num := NextNumber(cfg, state, "")
	t.Logf("Generated: %s", num)
	expected := "NF-0100"
	if num != expected {
		t.Errorf("expected %s, got %s", expected, num)
	}
}

func TestNextNumber_CustomSeparator(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:      "TB",
			StartNumber: 1,
			Padding:     5,
			Separator:   "/",
			IncludeYear: false,
		},
	}
	state := &config.State{NextInvoiceNumber: 7}

	num := NextNumber(cfg, state, "")
	t.Logf("Generated: %s", num)
	expected := "TB/00007"
	if num != expected {
		t.Errorf("expected %s, got %s", expected, num)
	}
}

func TestFormatNumber_Default(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:      "INV",
			Padding:     3,
			Separator:   "-",
			IncludeYear: true,
		},
	}
	year := time.Now().Year()
	got := FormatNumber(cfg, 5, "")
	want := fmt.Sprintf("INV-%d-005", year)
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestFormatNumber_PerClient(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:          "INV",
			Padding:         3,
			Separator:       "-",
			IncludeYear:     true,
			PerClientPrefix: true,
		},
	}
	year := time.Now().Year()
	got := FormatNumber(cfg, 42, "acme-corp")
	want := fmt.Sprintf("ACME-CORP-%d-042", year)
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestFormatNumber_NoYear(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:      "NF",
			Padding:     4,
			Separator:   "-",
			IncludeYear: false,
		},
	}
	got := FormatNumber(cfg, 7, "")
	want := "NF-0007"
	if got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestFormatNumber_DoesNotAdvanceState(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:      "INV",
			Padding:     3,
			Separator:   "-",
			IncludeYear: false,
		},
	}
	FormatNumber(cfg, 10, "")
	FormatNumber(cfg, 10, "")
	// No state to advance; just verifying determinism.
	got := FormatNumber(cfg, 10, "")
	if want := "INV-010"; got != want {
		t.Errorf("got %s, want %s", got, want)
	}
}

func TestPreview_DoesNotAdvance(t *testing.T) {
	cfg := &config.Config{
		Invoice: config.InvoiceSettings{
			Prefix:    "INV",
			Padding:   3,
			Separator: "-",
		},
	}
	state := &config.State{NextInvoiceNumber: 5}

	_ = Preview(cfg, state, "")
	if state.NextInvoiceNumber != 5 {
		t.Errorf("Preview should not advance counter, got %d", state.NextInvoiceNumber)
	}
}
