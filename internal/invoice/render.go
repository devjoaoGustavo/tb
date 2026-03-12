package invoice

import (
	"fmt"
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/devjoaoGustavo/tb/internal/i18n"
	"github.com/devjoaoGustavo/tb/internal/model"
)

// InvoiceData is the view model passed to the HTML template.
type InvoiceData struct {
	Issuer          model.Issuer
	Client          model.Client
	Project         model.Project
	Invoice         model.Invoice
	CurrSymbol      string
	FormattedDue    string
	FormattedIssued string
	FormattedPeriod string
	Locale          *i18n.Locale
}

// CurrencySymbol returns the display symbol for a currency code.
func CurrencySymbol(c model.Currency) string {
	switch c {
	case model.CurrencyBRL:
		return "R$"
	case model.CurrencyUSD:
		return "$"
	case model.CurrencyEUR:
		return "€"
	default:
		return string(c)
	}
}

// FormatMoney formats a float as a currency string.
func FormatMoney(amount float64, symbol string) string {
	return fmt.Sprintf("%s %.2f", symbol, amount)
}

// Render writes the invoice HTML to the provided writer.
func Render(w io.Writer, inv model.Invoice, project model.Project, client model.Client, issuer model.Issuer, locale string) error {
	loc := i18n.Get(locale)
	sym := CurrencySymbol(inv.Currency)

	data := InvoiceData{
		Issuer:          issuer,
		Client:          client,
		Project:         project,
		Invoice:         inv,
		CurrSymbol:      sym,
		FormattedDue:    i18n.FormatDate(inv.DueAt, loc.DateFormat),
		FormattedIssued: i18n.FormatDate(inv.IssuedAt, loc.DateFormat),
		Locale:          loc,
	}

	if !inv.PeriodStart.IsZero() && !inv.PeriodEnd.IsZero() {
		data.FormattedPeriod = fmt.Sprintf("%s — %s",
			i18n.FormatDate(inv.PeriodStart, loc.ShortDateFormat),
			i18n.FormatDate(inv.PeriodEnd, loc.DateFormat))
	}

	funcMap := template.FuncMap{
		"money": func(amount float64) string {
			return i18n.FormatMoney(amount, sym, loc)
		},
		"fmtHours": func(h float64) string {
			intPart := int(h)
			decPart := int((h - float64(intPart)) * 10)
			return fmt.Sprintf("%d%s%dh", intPart, loc.DecimalSep, decPart)
		},
		"upper": strings.ToUpper,
		"statusColor": func(s model.InvoiceStatus) string {
			switch s {
			case model.InvoicePaid:
				return "#16a34a"
			case model.InvoiceSent:
				return "#d97706"
			default:
				return "#6b7280"
			}
		},
		"isHourly": func() bool {
			return inv.BillingType == model.BillingHourly
		},
		"hasTax": func() bool {
			return inv.TaxRate > 0
		},
		"taxPercent": func() string {
			return fmt.Sprintf("%.0f%%", inv.TaxRate*100)
		},
		"loc": func() *i18n.Locale {
			return loc
		},
		"paymentDueMsg": func() template.HTML {
			msg := fmt.Sprintf(loc.PaymentDueBy, "<strong>"+data.FormattedDue+"</strong>")
			return template.HTML(msg)
		},
	}

	tmpl, err := template.New("invoice").Funcs(funcMap).Parse(invoiceTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	return tmpl.Execute(w, data)
}

// RenderToFile renders the invoice HTML to the file at path, creating parent directories as needed.
func RenderToFile(path string, inv model.Invoice, project model.Project, client model.Client, issuer model.Issuer, locale string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return Render(f, inv, project, client, issuer, locale)
}
