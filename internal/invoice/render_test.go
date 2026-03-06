package invoice

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/devjoaoGustavo/tb/internal/model"
)

var fixedTime = time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

func makeTestData() (model.Invoice, model.Project, model.Client, model.Issuer) {
	issuer := model.Issuer{
		Name:  "Jane Freelancer",
		Email: "jane@example.com",
	}

	client := model.Client{
		ID:   "acme",
		Name: "Acme Corp",
	}

	project := model.Project{
		ID:          "proj-1",
		Name:        "Website Redesign",
		ClientID:    client.ID,
		BillingType: model.BillingHourly,
		HourlyRate:  100.0,
		Currency:    model.CurrencyUSD,
		Active:      true,
	}

	inv := model.Invoice{
		ID:          "inv-1",
		Number:      "ACME-2026-001",
		ClientID:    client.ID,
		ProjectID:   project.ID,
		Status:      model.InvoiceDraft,
		BillingType: model.BillingHourly,
		Currency:    model.CurrencyUSD,
		LineItems: []model.LineItem{
			{
				Description: "Frontend development",
				Hours:       8.0,
				Rate:        100.0,
				Amount:      800.0,
			},
		},
		Subtotal: 800.0,
		Total:    800.0,
		IssuedAt: fixedTime,
		DueAt:    fixedTime.AddDate(0, 0, 30),
	}

	return inv, project, client, issuer
}

func TestCurrencySymbol(t *testing.T) {
	tests := []struct {
		currency model.Currency
		want     string
	}{
		{model.CurrencyBRL, "R$"},
		{model.CurrencyUSD, "$"},
		{model.CurrencyEUR, "€"},
		{"GBP", "GBP"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.currency), func(t *testing.T) {
			got := CurrencySymbol(tt.currency)
			if got != tt.want {
				t.Errorf("CurrencySymbol(%q) = %q, want %q", tt.currency, got, tt.want)
			}
		})
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		amount float64
		symbol string
		want   string
	}{
		{1234.50, "R$", "R$ 1234.50"},
		{0.0, "$", "$ 0.00"},
		{99.9, "€", "€ 99.90"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatMoney(tt.amount, tt.symbol)
			if got != tt.want {
				t.Errorf("FormatMoney(%.2f, %q) = %q, want %q", tt.amount, tt.symbol, got, tt.want)
			}
		})
	}
}

func TestRender_Hourly(t *testing.T) {
	inv, project, client, issuer := makeTestData()

	var buf bytes.Buffer
	if err := Render(&buf, inv, project, client, issuer); err != nil {
		t.Fatalf("Render: %v", err)
	}

	output := buf.String()

	for _, want := range []string{
		inv.Number,
		client.Name,
		issuer.Name,
		"Frontend development",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output does not contain %q", want)
		}
	}
}

func TestRender_Fixed(t *testing.T) {
	inv, project, client, issuer := makeTestData()

	inv.BillingType = model.BillingFixed
	project.BillingType = model.BillingFixed
	project.FixedAmount = 5000.0
	inv.LineItems = []model.LineItem{
		{
			Description: "Monthly retainer",
			Amount:      5000.0,
		},
	}
	inv.Subtotal = 5000.0
	inv.Total = 5000.0

	var buf bytes.Buffer
	if err := Render(&buf, inv, project, client, issuer); err != nil {
		t.Fatalf("Render: %v", err)
	}

	output := buf.String()

	for _, want := range []string{
		inv.Number,
		client.Name,
		issuer.Name,
		"Monthly retainer",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("output does not contain %q", want)
		}
	}
}

func TestRenderToFile(t *testing.T) {
	inv, project, client, issuer := makeTestData()

	dir := t.TempDir()
	path := filepath.Join(dir, "invoices", "2026", "invoice.html")

	if err := RenderToFile(path, inv, project, client, issuer); err != nil {
		t.Fatalf("RenderToFile: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	output := string(data)

	for _, want := range []string{
		inv.Number,
		client.Name,
		issuer.Name,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("file content does not contain %q", want)
		}
	}
}
