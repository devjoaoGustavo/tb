package i18n

import (
	"fmt"
	"math"
	"strings"
	"time"
)

// Locale holds all translatable strings used in invoices.
type Locale struct {
	// Header labels
	InvoiceLabel string
	BillTo       string
	Issued       string
	DueDate      string
	Period       string
	Currency     string
	Project      string

	// Billing type labels
	Hourly     string
	FixedPrice string

	// Table headers
	Date        string
	Description string
	Hours       string
	Rate        string
	Amount      string

	// Totals
	Subtotal string
	Tax      string
	TotalDue string

	// Other sections
	Notes string

	// Footer messages
	ThankYou      string
	PaymentDueBy  string // contains %s placeholder for date
	GeneratedWith string

	// Invoice statuses
	StatusDraft string
	StatusSent  string
	StatusPaid  string

	// Date formats (Go time layout strings)
	DateFormat      string
	ShortDateFormat string

	// Money formatting
	MoneyFormat  string // "prefix" or "suffix"
	DecimalSep   string
	ThousandsSep string
}

var locales = map[string]*Locale{
	"en": {
		InvoiceLabel:    "Invoice",
		BillTo:          "Bill To",
		Issued:          "Issued",
		DueDate:         "Due Date",
		Period:          "Period",
		Currency:        "Currency",
		Project:         "Project",
		Hourly:          "Hourly",
		FixedPrice:      "Fixed Price",
		Date:            "Date",
		Description:     "Description",
		Hours:           "Hours",
		Rate:            "Rate",
		Amount:          "Amount",
		Subtotal:        "Subtotal",
		Tax:             "Tax",
		TotalDue:        "Total Due",
		Notes:           "Notes",
		ThankYou:        "Thank you for your business.",
		PaymentDueBy:    "Payment is due by %s.",
		GeneratedWith:   "generated with",
		StatusDraft:     "Draft",
		StatusSent:      "Sent",
		StatusPaid:      "Paid",
		DateFormat:      "January 02, 2006",
		ShortDateFormat: "Jan 02",
		MoneyFormat:     "prefix",
		DecimalSep:      ".",
		ThousandsSep:    ",",
	},
	"es": {
		InvoiceLabel:    "Factura",
		BillTo:          "Facturar A",
		Issued:          "Emitido",
		DueDate:         "Fecha de Vencimiento",
		Period:          "Período",
		Currency:        "Moneda",
		Project:         "Proyecto",
		Hourly:          "Por Hora",
		FixedPrice:      "Precio Fijo",
		Date:            "Fecha",
		Description:     "Descripción",
		Hours:           "Horas",
		Rate:            "Tarifa",
		Amount:          "Importe",
		Subtotal:        "Subtotal",
		Tax:             "Impuesto",
		TotalDue:        "Total a Pagar",
		Notes:           "Notas",
		ThankYou:        "Gracias por su confianza.",
		PaymentDueBy:    "El pago vence el %s.",
		GeneratedWith:   "generado con",
		StatusDraft:     "Borrador",
		StatusSent:      "Enviada",
		StatusPaid:      "Pagada",
		DateFormat:      "02 de January de 2006",
		ShortDateFormat: "02 Jan",
		MoneyFormat:     "prefix",
		DecimalSep:      ",",
		ThousandsSep:    ".",
	},
	"it": {
		InvoiceLabel:    "Fattura",
		BillTo:          "Intestato A",
		Issued:          "Emesso",
		DueDate:         "Data di Scadenza",
		Period:          "Periodo",
		Currency:        "Valuta",
		Project:         "Progetto",
		Hourly:          "A Ore",
		FixedPrice:      "Prezzo Fisso",
		Date:            "Data",
		Description:     "Descrizione",
		Hours:           "Ore",
		Rate:            "Tariffa",
		Amount:          "Importo",
		Subtotal:        "Subtotale",
		Tax:             "Imposta",
		TotalDue:        "Totale Dovuto",
		Notes:           "Note",
		ThankYou:        "Grazie per la fiducia.",
		PaymentDueBy:    "Il pagamento è dovuto entro il %s.",
		GeneratedWith:   "generato con",
		StatusDraft:     "Bozza",
		StatusSent:      "Inviata",
		StatusPaid:      "Pagata",
		DateFormat:      "02 January 2006",
		ShortDateFormat: "02 Jan",
		MoneyFormat:     "prefix",
		DecimalSep:      ",",
		ThousandsSep:    ".",
	},
	"pt-BR": {
		InvoiceLabel:    "Fatura",
		BillTo:          "Cobrar De",
		Issued:          "Emitido",
		DueDate:         "Data de Vencimento",
		Period:          "Período",
		Currency:        "Moeda",
		Project:         "Projeto",
		Hourly:          "Por Hora",
		FixedPrice:      "Preço Fixo",
		Date:            "Data",
		Description:     "Descrição",
		Hours:           "Horas",
		Rate:            "Tarifa",
		Amount:          "Valor",
		Subtotal:        "Subtotal",
		Tax:             "Imposto",
		TotalDue:        "Total a Pagar",
		Notes:           "Notas",
		ThankYou:        "Obrigado pela confiança.",
		PaymentDueBy:    "O pagamento vence em %s.",
		GeneratedWith:   "gerado com",
		StatusDraft:     "Rascunho",
		StatusSent:      "Enviada",
		StatusPaid:      "Paga",
		DateFormat:      "02 de January de 2006",
		ShortDateFormat: "02 Jan",
		MoneyFormat:     "prefix",
		DecimalSep:      ",",
		ThousandsSep:    ".",
	},
	"pt": {
		InvoiceLabel:    "Fatura",
		BillTo:          "Faturar A",
		Issued:          "Emitido",
		DueDate:         "Data de Vencimento",
		Period:          "Período",
		Currency:        "Moeda",
		Project:         "Projeto",
		Hourly:          "Por Hora",
		FixedPrice:      "Preço Fixo",
		Date:            "Data",
		Description:     "Descrição",
		Hours:           "Horas",
		Rate:            "Tarifa",
		Amount:          "Valor",
		Subtotal:        "Subtotal",
		Tax:             "Imposto",
		TotalDue:        "Total a Pagar",
		Notes:           "Notas",
		ThankYou:        "Obrigado pela preferência.",
		PaymentDueBy:    "O pagamento vence em %s.",
		GeneratedWith:   "gerado com",
		StatusDraft:     "Rascunho",
		StatusSent:      "Enviada",
		StatusPaid:      "Paga",
		DateFormat:      "02 de January de 2006",
		ShortDateFormat: "02 Jan",
		MoneyFormat:     "prefix",
		DecimalSep:      ",",
		ThousandsSep:    ".",
	},
}

// Get returns the Locale for the given locale code. Falls back to "en" if unknown.
func Get(localeCode string) *Locale {
	if loc, ok := locales[localeCode]; ok {
		return loc
	}
	// Try matching just the language part (e.g., "pt-PT" falls back to "pt")
	if idx := strings.IndexByte(localeCode, '-'); idx > 0 {
		if loc, ok := locales[localeCode[:idx]]; ok {
			return loc
		}
	}
	return locales["en"]
}

// FormatMoney formats a monetary amount according to the locale's separators
// and prefix/suffix pattern. The symbol is the currency symbol (e.g., "$", "R$").
func FormatMoney(amount float64, symbol string, loc *Locale) string {
	negative := amount < 0
	amount = math.Abs(amount)

	// Split into integer and decimal parts
	intPart := int64(amount)
	decPart := int64(math.Round((amount - float64(intPart)) * 100))

	// Format integer part with thousands separator
	intStr := formatIntWithSep(intPart, loc.ThousandsSep)

	// Build the number string
	numStr := fmt.Sprintf("%s%s%02d", intStr, loc.DecimalSep, decPart)

	if negative {
		numStr = "-" + numStr
	}

	// Apply prefix or suffix pattern
	if loc.MoneyFormat == "suffix" {
		return numStr + " " + symbol
	}
	return symbol + " " + numStr
}

// formatIntWithSep formats an integer with a thousands separator.
func formatIntWithSep(n int64, sep string) string {
	if sep == "" {
		return fmt.Sprintf("%d", n)
	}

	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result strings.Builder
	remainder := len(s) % 3
	if remainder > 0 {
		result.WriteString(s[:remainder])
	}
	for i := remainder; i < len(s); i += 3 {
		if result.Len() > 0 {
			result.WriteString(sep)
		}
		result.WriteString(s[i : i+3])
	}
	return result.String()
}

// FormatDate formats a time value using the given Go time layout string.
func FormatDate(t time.Time, format string) string {
	return t.Format(format)
}
