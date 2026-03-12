// Package ui provides TUI styling helpers for the tb CLI.
package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// Color theme constants.
const (
	ColorAccent  = lipgloss.Color("#2563eb")
	ColorSuccess = lipgloss.Color("#16a34a")
	ColorWarning = lipgloss.Color("#d97706")
	ColorError   = lipgloss.Color("#dc2626")
	ColorMuted   = lipgloss.Color("#8a8a8a")
	ColorSubtle  = lipgloss.Color("#4a4a4a")
)

// Unicode indicators.
const (
	Checkmark = "✓"
	Cross     = "✗"
)

// Pre-built styles.
var (
	Title    = lipgloss.NewStyle().Bold(true).Foreground(ColorAccent)
	Subtitle = lipgloss.NewStyle().Bold(true).Foreground(ColorSubtle)
	Success  = lipgloss.NewStyle().Bold(true).Foreground(ColorSuccess)
	Warning  = lipgloss.NewStyle().Foreground(ColorWarning)
	Error    = lipgloss.NewStyle().Bold(true).Foreground(ColorError)
	Muted    = lipgloss.NewStyle().Foreground(ColorMuted)
	Keyword  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#ffffff"))
)

// StatusBadge returns a styled status indicator string.
func StatusBadge(status string) string {
	switch strings.ToLower(status) {
	case "draft":
		return Muted.Render("● Draft")
	case "sent":
		return Warning.Render("● Sent")
	case "paid":
		return Success.Render("● Paid")
	case "overdue":
		return Error.Render("● Overdue")
	default:
		return Muted.Render("● " + status)
	}
}

// RenderTable renders a styled table with headers and rows.
func RenderTable(headers []string, rows [][]string) string {
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Padding(0, 1)

	evenRowStyle := lipgloss.NewStyle().
		Padding(0, 1)

	oddRowStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Background(ColorSubtle)

	t := table.New().
		Headers(headers...).
		Rows(rows...).
		Border(lipgloss.NormalBorder()).
		BorderHeader(true).
		BorderColumn(true).
		BorderRow(false).
		BorderTop(true).
		BorderBottom(true).
		BorderLeft(true).
		BorderRight(true).
		BorderStyle(lipgloss.NewStyle().Foreground(ColorSubtle)).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			if row%2 == 0 {
				return evenRowStyle
			}
			return oddRowStyle
		})

	return t.String()
}

// SectionHeader renders a section header like: ─── TITLE ─────────────
func SectionHeader(title string) string {
	label := strings.ToUpper(title)
	prefix := "─── "
	suffix := " " + strings.Repeat("─", max(0, 30-len(label)))
	line := prefix + label + suffix
	return Title.Render(line)
}

// KeyValue renders a key-value pair with the key in muted style and value in keyword style.
func KeyValue(key, value string) string {
	return Muted.Render(key+":") + "  " + Keyword.Render(value)
}

// Banner renders a banner with a title and subtitle for the root help/header.
func Banner(title, subtitle string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(ColorAccent).
		Padding(0, 1)

	subtitleStyle := lipgloss.NewStyle().
		Foreground(ColorMuted).
		Padding(0, 1)

	border := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(ColorAccent).
		Padding(0, 2)

	content := titleStyle.Render(title) + "\n" + subtitleStyle.Render(subtitle)
	return border.Render(content)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

