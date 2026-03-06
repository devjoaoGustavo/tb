package numbering

import (
	"fmt"
	"strings"
	"time"

	"github.com/devjoaoGustavo/tb/internal/config"
)

// NextNumber generates the next invoice number string and advances the counter.
//
// Examples with different configs:
//   Default:                 "INV-2026-001"
//   PerClientPrefix=true:    "ACME-2026-001"
//   IncludeYear=false:       "INV-001"
//   Padding=4, Start=100:    "INV-2026-0100"
func NextNumber(cfg *config.Config, state *config.State, clientSlug string) string {
	settings := cfg.Invoice
	sep := settings.Separator

	prefix := settings.Prefix
	if settings.PerClientPrefix && clientSlug != "" {
		prefix = strings.ToUpper(clientSlug)
	}

	numStr := fmt.Sprintf("%0*d", settings.Padding, state.NextInvoiceNumber)

	var parts []string
	parts = append(parts, prefix)
	if settings.IncludeYear {
		parts = append(parts, fmt.Sprintf("%d", time.Now().Year()))
	}
	parts = append(parts, numStr)

	state.NextInvoiceNumber++

	return strings.Join(parts, sep)
}

// Preview returns what the next invoice number would look like without advancing.
func Preview(cfg *config.Config, state *config.State, clientSlug string) string {
	stateCopy := &config.State{NextInvoiceNumber: state.NextInvoiceNumber}
	return NextNumber(cfg, stateCopy, clientSlug)
}

// FormatNumber formats a specific number using the current config rules.
func FormatNumber(cfg *config.Config, num int, clientSlug string) string {
	settings := cfg.Invoice
	sep := settings.Separator

	prefix := settings.Prefix
	if settings.PerClientPrefix && clientSlug != "" {
		prefix = strings.ToUpper(clientSlug)
	}

	numStr := fmt.Sprintf("%0*d", settings.Padding, num)

	var parts []string
	parts = append(parts, prefix)
	if settings.IncludeYear {
		parts = append(parts, fmt.Sprintf("%d", time.Now().Year()))
	}
	parts = append(parts, numStr)

	return strings.Join(parts, sep)
}
