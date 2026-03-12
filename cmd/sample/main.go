package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/devjoaoGustavo/tb/internal/config"
	"github.com/devjoaoGustavo/tb/internal/invoice"
	"github.com/devjoaoGustavo/tb/internal/model"
	"github.com/devjoaoGustavo/tb/internal/numbering"
)

func main() {
	cfg := config.DefaultConfig()
	cfg.Issuer = model.Issuer{
		Name:    "Jo\u00e3o Gustavo",
		Title:   "Software Engineer & Consultant",
		Email:   "jgustavo@vio.com",
		Phone:   "+55 11 99999-0000",
		Address: "Vargem Grande Paulista, SP \u2014 Brasil",
		TaxID:   "12.345.678/0001-90",
	}
	cfg.Invoice.Prefix = "INV"
	cfg.Invoice.StartNumber = 1
	cfg.Invoice.Padding = 3
	cfg.Invoice.IncludeYear = true
	cfg.Invoice.DefaultDueDays = 15
	cfg.Invoice.DefaultTaxRate = 0.05
	cfg.Invoice.DefaultCurrency = model.CurrencyBRL
	cfg.Invoice.DefaultNotes = "Payment via PIX (CNPJ key) or bank transfer."

	cfgJSON, _ := json.MarshalIndent(cfg, "", "  ")
	fmt.Println("~/.config/tb/config.json")
	fmt.Println(string(cfgJSON))
	fmt.Println()

	state := &config.State{NextInvoiceNumber: cfg.Invoice.StartNumber}

	generateHourly(&cfg, state)
	generateFixed(&cfg, state)

	stateJSON, _ := json.MarshalIndent(state, "", "  ")
	fmt.Println("\n~/.local/share/tb/state.json")
	fmt.Println(string(stateJSON))

	preview := numbering.Preview(&cfg, state, "")
	fmt.Printf("\nNext invoice would be: %s\n", preview)
	fmt.Printf("Counter still at: %d (not advanced)\n", state.NextInvoiceNumber)

	cfgClient := cfg
	cfgClient.Invoice.PerClientPrefix = true
	stateClient := &config.State{NextInvoiceNumber: 1}
	fmt.Println("\nPer-client prefix examples:")
	fmt.Printf("  %s\n", numbering.NextNumber(&cfgClient, stateClient, "acme"))
	fmt.Printf("  %s\n", numbering.NextNumber(&cfgClient, stateClient, "acme"))
	fmt.Printf("  %s\n", numbering.NextNumber(&cfgClient, stateClient, "globex"))
}

func generateHourly(cfg *config.Config, state *config.State) {
	client := model.Client{
		ID:      "techstart",
		Name:    "TechStart Inc.",
		Company: "TechStart Tecnologia Ltda",
		Email:   "finance@techstart.com.br",
		Address: "Av. Paulista, 1000 \u2014 S\u00e3o Paulo, SP",
		TaxID:   "12.345.678/0001-90",
	}
	project := model.Project{
		ID:          "api-migration",
		Name:        "API Migration \u2014 v2 Rewrite",
		ClientID:    "techstart",
		BillingType: model.BillingHourly,
		HourlyRate:  150,
		Currency:    model.CurrencyBRL,
	}
	invNumber := numbering.NextNumber(cfg, state, client.ID)
	subtotal := 5700.0
	inv := model.Invoice{
		Number:      invNumber,
		ClientID:    "techstart",
		ProjectID:   "api-migration",
		Status:      model.InvoiceSent,
		BillingType: model.BillingHourly,
		Currency:    model.CurrencyBRL,
		IssuedAt:    time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		DueAt:       time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC).AddDate(0, 0, cfg.Invoice.DefaultDueDays),
		PeriodStart: time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 2, 28, 0, 0, 0, 0, time.UTC),
		LineItems: []model.LineItem{
			{Date: "Feb 03", Description: "Authentication module refactor", Hours: 4.5, Rate: 150, Amount: 675},
			{Date: "Feb 05", Description: "Database schema migration scripts", Hours: 6.0, Rate: 150, Amount: 900},
			{Date: "Feb 07", Description: "REST endpoints users and permissions", Hours: 5.5, Rate: 150, Amount: 825},
			{Date: "Feb 10", Description: "Integration tests for booking flow", Hours: 3.0, Rate: 150, Amount: 450},
			{Date: "Feb 12", Description: "WebSocket event pipeline setup", Hours: 7.0, Rate: 150, Amount: 1050},
			{Date: "Feb 17", Description: "Code review + pair programming", Hours: 2.5, Rate: 150, Amount: 375},
			{Date: "Feb 19", Description: "CI/CD pipeline adjustments", Hours: 3.5, Rate: 150, Amount: 525},
			{Date: "Feb 24", Description: "Performance profiling and optimization", Hours: 4.0, Rate: 150, Amount: 600},
			{Date: "Feb 26", Description: "Documentation and handoff notes", Hours: 2.0, Rate: 150, Amount: 300},
		},
		Subtotal: subtotal,
		TaxRate:  cfg.Invoice.DefaultTaxRate,
		Tax:      subtotal * cfg.Invoice.DefaultTaxRate,
		Total:    subtotal + (subtotal * cfg.Invoice.DefaultTaxRate),
		Notes:    cfg.Invoice.DefaultNotes + " Reference: " + invNumber,
	}
	f, _ := os.Create("invoice-hourly.html")
	defer f.Close()
	invoice.Render(f, inv, project, client, cfg.Issuer, cfg.Locale)
	fmt.Printf("Generated invoice-hourly.html  [%s]\n", invNumber)
}

func generateFixed(cfg *config.Config, state *config.State) {
	client := model.Client{
		ID:      "globex",
		Name:    "Maria Silva",
		Company: "Globex Solutions",
		Email:   "maria@globex.io",
		Address: "R. Augusta, 500 \u2014 S\u00e3o Paulo, SP",
	}
	project := model.Project{
		ID:          "landing-page",
		Name:        "Globex \u2014 Landing Page Redesign",
		ClientID:    "globex",
		BillingType: model.BillingFixed,
		FixedAmount: 8000,
		Currency:    model.CurrencyBRL,
	}
	invNumber := numbering.NextNumber(cfg, state, client.ID)
	inv := model.Invoice{
		Number:      invNumber,
		ClientID:    "globex",
		ProjectID:   "landing-page",
		Status:      model.InvoicePaid,
		BillingType: model.BillingFixed,
		Currency:    model.CurrencyBRL,
		IssuedAt:    time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC),
		DueAt:       time.Date(2026, 2, 10, 0, 0, 0, 0, time.UTC).AddDate(0, 0, cfg.Invoice.DefaultDueDays),
		PaidAt:      time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC),
		LineItems: []model.LineItem{
			{Description: "Discovery and wireframes (milestone 1)", Amount: 2000},
			{Description: "UI design and responsive layouts (milestone 2)", Amount: 3000},
			{Description: "Development and deployment (milestone 3)", Amount: 3000},
		},
		Subtotal: 8000,
		Total:    8000,
		Notes:    "Project delivered and approved. Thank you!",
	}
	f, _ := os.Create("invoice-fixed.html")
	defer f.Close()
	invoice.Render(f, inv, project, client, cfg.Issuer, cfg.Locale)
	fmt.Printf("Generated invoice-fixed.html   [%s]\n", invNumber)
}
