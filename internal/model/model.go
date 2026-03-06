package model

import "time"

type BillingType string

const (
	BillingHourly BillingType = "hourly"
	BillingFixed  BillingType = "fixed"
)

type InvoiceStatus string

const (
	InvoiceDraft InvoiceStatus = "draft"
	InvoiceSent  InvoiceStatus = "sent"
	InvoicePaid  InvoiceStatus = "paid"
)

type Currency string

const (
	CurrencyBRL Currency = "BRL"
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
)

// Client represents a billable client.
type Client struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Email    string `json:"email,omitempty"`
	Company  string `json:"company,omitempty"`
	Address  string `json:"address,omitempty"`
	TaxID    string `json:"tax_id,omitempty"` // CPF/CNPJ for Brazil
	Phone    string `json:"phone,omitempty"`
	Created  time.Time `json:"created"`
}

// Project represents a body of work for a client.
type Project struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	ClientID    string      `json:"client_id"`
	BillingType BillingType `json:"billing_type"`
	HourlyRate  float64     `json:"hourly_rate,omitempty"`  // used when billing_type == "hourly"
	FixedAmount float64     `json:"fixed_amount,omitempty"` // used when billing_type == "fixed"
	Currency    Currency    `json:"currency"`
	Active      bool        `json:"active"`
	Created     time.Time   `json:"created"`
}

// Session represents a tracked block of time.
type Session struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	Start     time.Time `json:"start"`
	End       time.Time `json:"end,omitempty"` // zero value means still running
	Note      string    `json:"note,omitempty"`
	Tags      []string  `json:"tags,omitempty"`
	Billed    bool      `json:"billed"` // already included in an invoice
}

// Duration returns the session length. If still running, uses time.Now().
func (s Session) Duration() time.Duration {
	end := s.End
	if end.IsZero() {
		end = time.Now()
	}
	return end.Sub(s.Start)
}

// Hours returns the duration as a decimal number of hours.
func (s Session) Hours() float64 {
	return s.Duration().Hours()
}

// Invoice is the generated billing document.
type Invoice struct {
	ID          string        `json:"id"`
	Number      string        `json:"number"`      // e.g. "ACME-2026-003"
	ClientID    string        `json:"client_id"`
	ProjectID   string        `json:"project_id"`
	Status      InvoiceStatus `json:"status"`
	BillingType BillingType   `json:"billing_type"`
	Currency    Currency      `json:"currency"`
	LineItems   []LineItem    `json:"line_items"`
	Subtotal    float64       `json:"subtotal"`
	Tax         float64       `json:"tax,omitempty"`
	TaxRate     float64       `json:"tax_rate,omitempty"` // e.g. 0.05 for 5%
	Total       float64       `json:"total"`
	IssuedAt    time.Time     `json:"issued_at"`
	DueAt       time.Time     `json:"due_at"`
	PaidAt      time.Time     `json:"paid_at,omitempty"`
	Notes       string        `json:"notes,omitempty"`
	PeriodStart time.Time     `json:"period_start,omitempty"`
	PeriodEnd   time.Time     `json:"period_end,omitempty"`
	Created     time.Time     `json:"created"`
}

// LineItem is a single row in an invoice.
type LineItem struct {
	Description string  `json:"description"`
	Date        string  `json:"date,omitempty"`
	Hours       float64 `json:"hours,omitempty"`      // for hourly billing
	Rate        float64 `json:"rate,omitempty"`       // per-hour rate
	Amount      float64 `json:"amount"`               // final line amount
	SessionID   string  `json:"session_id,omitempty"` // source session, used to unmark on invoice delete
}

// Issuer holds the freelancer's own info for the invoice header.
type Issuer struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Email   string `json:"email"`
	Phone   string `json:"phone,omitempty"`
	Address string `json:"address,omitempty"`
	TaxID   string `json:"tax_id,omitempty"` // CPF/CNPJ
	Website string `json:"website,omitempty"`
	LogoURL string `json:"logo_url,omitempty"`
}
