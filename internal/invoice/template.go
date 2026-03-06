package invoice

const invoiceTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Invoice {{ .Invoice.Number }}</title>
<link rel="preconnect" href="https://fonts.googleapis.com">
<link href="https://fonts.googleapis.com/css2?family=DM+Mono:wght@400;500&family=DM+Sans:wght@400;500;700&display=swap" rel="stylesheet">
<style>
  :root {
    --ink: #1a1a1a;
    --ink-soft: #4a4a4a;
    --ink-muted: #8a8a8a;
    --surface: #ffffff;
    --surface-alt: #f7f7f5;
    --border: #e5e5e3;
    --accent: #2563eb;
    --accent-soft: #eff4ff;
    --paid: #16a34a;
    --pending: #d97706;
    --font-body: 'DM Sans', -apple-system, sans-serif;
    --font-mono: 'DM Mono', 'SF Mono', monospace;
  }

  * { margin: 0; padding: 0; box-sizing: border-box; }

  @page {
    size: A4;
    margin: 0;
  }

  body {
    font-family: var(--font-body);
    color: var(--ink);
    background: var(--surface);
    font-size: 14px;
    line-height: 1.6;
    -webkit-font-smoothing: antialiased;
  }

  .invoice-page {
    width: 210mm;
    min-height: 297mm;
    margin: 0 auto;
    padding: 48px 56px;
    position: relative;
    display: flex;
    flex-direction: column;
  }

  /* ── Top accent bar ── */
  .invoice-page::before {
    content: '';
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    height: 5px;
    background: linear-gradient(90deg, var(--accent) 0%, #7c3aed 100%);
  }

  /* ── Header ── */
  .header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    padding-bottom: 40px;
    border-bottom: 1px solid var(--border);
  }

  .issuer-block h1 {
    font-size: 22px;
    font-weight: 700;
    letter-spacing: -0.5px;
    margin-bottom: 4px;
  }

  .issuer-block .title {
    font-size: 13px;
    color: var(--ink-muted);
    margin-bottom: 12px;
  }

  .issuer-block .contact-line {
    font-size: 12px;
    color: var(--ink-soft);
    line-height: 1.8;
  }

  .invoice-id-block {
    text-align: right;
  }

  .invoice-label {
    font-family: var(--font-mono);
    font-size: 11px;
    letter-spacing: 2px;
    color: var(--ink-muted);
    text-transform: uppercase;
    margin-bottom: 6px;
  }

  .invoice-number {
    font-family: var(--font-mono);
    font-size: 28px;
    font-weight: 500;
    letter-spacing: -1px;
    color: var(--accent);
  }

  .invoice-status {
    display: inline-block;
    margin-top: 10px;
    font-family: var(--font-mono);
    font-size: 11px;
    letter-spacing: 1.5px;
    text-transform: uppercase;
    padding: 4px 12px;
    border-radius: 4px;
    font-weight: 500;
  }

  .status-draft {
    background: var(--surface-alt);
    color: var(--ink-muted);
    border: 1px solid var(--border);
  }

  .status-sent {
    background: #fffbeb;
    color: var(--pending);
    border: 1px solid #fde68a;
  }

  .status-paid {
    background: #f0fdf4;
    color: var(--paid);
    border: 1px solid #bbf7d0;
  }

  /* ── Meta row (Bill To + Dates) ── */
  .meta-row {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: 32px;
    padding: 32px 0;
  }

  .meta-group {}

  .meta-label {
    font-family: var(--font-mono);
    font-size: 10px;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--ink-muted);
    margin-bottom: 8px;
  }

  .meta-value {
    font-size: 14px;
    color: var(--ink);
    line-height: 1.7;
  }

  .meta-value strong {
    font-weight: 700;
    font-size: 15px;
  }

  .meta-value .small {
    font-size: 12px;
    color: var(--ink-soft);
  }

  /* ── Project banner ── */
  .project-banner {
    background: var(--surface-alt);
    border: 1px solid var(--border);
    border-radius: 8px;
    padding: 16px 24px;
    margin-bottom: 32px;
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .project-banner .project-name {
    font-weight: 700;
    font-size: 15px;
  }

  .project-banner .project-type {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--ink-muted);
    letter-spacing: 1px;
    text-transform: uppercase;
    background: var(--surface);
    padding: 4px 10px;
    border-radius: 4px;
    border: 1px solid var(--border);
  }

  /* ── Line Items Table ── */
  .items-table {
    width: 100%;
    border-collapse: collapse;
    margin-bottom: 32px;
  }

  .items-table thead th {
    font-family: var(--font-mono);
    font-size: 10px;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--ink-muted);
    padding: 12px 16px;
    text-align: left;
    border-bottom: 2px solid var(--ink);
  }

  .items-table thead th:last-child,
  .items-table thead th.num {
    text-align: right;
  }

  .items-table tbody td {
    padding: 14px 16px;
    font-size: 13px;
    border-bottom: 1px solid var(--border);
    vertical-align: top;
  }

  .items-table tbody td.num {
    text-align: right;
    font-family: var(--font-mono);
    font-size: 13px;
    white-space: nowrap;
  }

  .items-table tbody td.desc {
    color: var(--ink-soft);
    max-width: 300px;
  }

  .items-table tbody td.date-col {
    font-family: var(--font-mono);
    font-size: 12px;
    color: var(--ink-muted);
    white-space: nowrap;
  }

  .items-table tbody tr:last-child td {
    border-bottom: 2px solid var(--ink);
  }

  /* ── Totals ── */
  .totals-section {
    display: flex;
    justify-content: flex-end;
    margin-bottom: 40px;
  }

  .totals-box {
    width: 280px;
  }

  .totals-row {
    display: flex;
    justify-content: space-between;
    padding: 8px 0;
    font-size: 13px;
    color: var(--ink-soft);
  }

  .totals-row .label {
    font-family: var(--font-mono);
    font-size: 11px;
    letter-spacing: 1px;
    text-transform: uppercase;
    color: var(--ink-muted);
  }

  .totals-row .value {
    font-family: var(--font-mono);
    font-size: 14px;
    color: var(--ink);
  }

  .totals-row.total-final {
    border-top: 2px solid var(--ink);
    margin-top: 8px;
    padding-top: 14px;
  }

  .totals-row.total-final .label {
    font-size: 13px;
    font-weight: 700;
    color: var(--ink);
    letter-spacing: 2px;
  }

  .totals-row.total-final .value {
    font-size: 20px;
    font-weight: 700;
    color: var(--accent);
  }

  /* ── Notes ── */
  .notes-section {
    background: var(--surface-alt);
    border-radius: 8px;
    padding: 20px 24px;
    margin-bottom: 40px;
    border: 1px solid var(--border);
  }

  .notes-section .notes-label {
    font-family: var(--font-mono);
    font-size: 10px;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--ink-muted);
    margin-bottom: 8px;
  }

  .notes-section p {
    font-size: 13px;
    color: var(--ink-soft);
    line-height: 1.7;
  }

  /* ── Footer ── */
  .footer {
    margin-top: auto;
    padding-top: 32px;
    border-top: 1px solid var(--border);
    display: flex;
    justify-content: space-between;
    align-items: flex-end;
  }

  .footer-left {
    font-size: 11px;
    color: var(--ink-muted);
    line-height: 1.8;
  }

  .footer-right {
    text-align: right;
  }

  .footer-brand {
    font-family: var(--font-mono);
    font-size: 11px;
    color: var(--ink-muted);
    letter-spacing: 1px;
  }

  .footer-brand span {
    color: var(--accent);
    font-weight: 500;
  }

  /* ── Print ── */
  @media print {
    body { background: white; }
    .invoice-page {
      padding: 40px 48px;
      width: 100%;
      min-height: 100vh;
    }
  }
</style>
</head>
<body>
<div class="invoice-page">

  <!-- Header -->
  <div class="header">
    <div class="issuer-block">
      <h1>{{ .Issuer.Name }}</h1>
      {{ if .Issuer.Title }}<div class="title">{{ .Issuer.Title }}</div>{{ end }}
      <div class="contact-line">
        {{ .Issuer.Email }}<br>
        {{ if .Issuer.Phone }}{{ .Issuer.Phone }}<br>{{ end }}
        {{ if .Issuer.Address }}{{ .Issuer.Address }}<br>{{ end }}
        {{ if .Issuer.TaxID }}Tax ID: {{ .Issuer.TaxID }}{{ end }}
      </div>
    </div>
    <div class="invoice-id-block">
      <div class="invoice-label">Invoice</div>
      <div class="invoice-number">{{ .Invoice.Number }}</div>
      <div>
        {{ if eq (print .Invoice.Status) "paid" }}
          <span class="invoice-status status-paid">● Paid</span>
        {{ else if eq (print .Invoice.Status) "sent" }}
          <span class="invoice-status status-sent">● Sent</span>
        {{ else }}
          <span class="invoice-status status-draft">● Draft</span>
        {{ end }}
      </div>
    </div>
  </div>

  <!-- Meta: Bill To + Dates -->
  <div class="meta-row">
    <div class="meta-group">
      <div class="meta-label">Bill To</div>
      <div class="meta-value">
        <strong>{{ .Client.Name }}</strong><br>
        {{ if .Client.Company }}{{ .Client.Company }}<br>{{ end }}
        {{ if .Client.Email }}<span class="small">{{ .Client.Email }}</span><br>{{ end }}
        {{ if .Client.Address }}<span class="small">{{ .Client.Address }}</span><br>{{ end }}
        {{ if .Client.TaxID }}<span class="small">Tax ID: {{ .Client.TaxID }}</span>{{ end }}
      </div>
    </div>
    <div class="meta-group">
      <div class="meta-label">Issued</div>
      <div class="meta-value">{{ .FormattedIssued }}</div>
      <br>
      <div class="meta-label">Due Date</div>
      <div class="meta-value"><strong>{{ .FormattedDue }}</strong></div>
    </div>
    <div class="meta-group">
      {{ if .FormattedPeriod }}
      <div class="meta-label">Period</div>
      <div class="meta-value">{{ .FormattedPeriod }}</div>
      <br>
      {{ end }}
      <div class="meta-label">Currency</div>
      <div class="meta-value">{{ .Invoice.Currency }} ({{ .CurrSymbol }})</div>
    </div>
  </div>

  <!-- Project Banner -->
  <div class="project-banner">
    <span class="project-name">{{ .Project.Name }}</span>
    <span class="project-type">
      {{ if isHourly }}⏱ Hourly{{ else }}📌 Fixed Price{{ end }}
    </span>
  </div>

  <!-- Line Items -->
  <table class="items-table">
    <thead>
      <tr>
        {{ if isHourly }}<th>Date</th>{{ end }}
        <th>Description</th>
        {{ if isHourly }}<th class="num">Hours</th>{{ end }}
        {{ if isHourly }}<th class="num">Rate</th>{{ end }}
        <th class="num">Amount</th>
      </tr>
    </thead>
    <tbody>
      {{ range .Invoice.LineItems }}
      <tr>
        {{ if isHourly }}<td class="date-col">{{ .Date }}</td>{{ end }}
        <td class="desc">{{ .Description }}</td>
        {{ if isHourly }}<td class="num">{{ fmtHours .Hours }}</td>{{ end }}
        {{ if isHourly }}<td class="num">{{ money .Rate }}</td>{{ end }}
        <td class="num">{{ money .Amount }}</td>
      </tr>
      {{ end }}
    </tbody>
  </table>

  <!-- Totals -->
  <div class="totals-section">
    <div class="totals-box">
      <div class="totals-row">
        <span class="label">Subtotal</span>
        <span class="value">{{ money .Invoice.Subtotal }}</span>
      </div>
      {{ if hasTax }}
      <div class="totals-row">
        <span class="label">Tax ({{ taxPercent }})</span>
        <span class="value">{{ money .Invoice.Tax }}</span>
      </div>
      {{ end }}
      <div class="totals-row total-final">
        <span class="label">Total Due</span>
        <span class="value">{{ money .Invoice.Total }}</span>
      </div>
    </div>
  </div>

  <!-- Notes -->
  {{ if .Invoice.Notes }}
  <div class="notes-section">
    <div class="notes-label">Notes</div>
    <p>{{ .Invoice.Notes }}</p>
  </div>
  {{ end }}

  <!-- Footer -->
  <div class="footer">
    <div class="footer-left">
      Thank you for your business.<br>
      Payment is due by <strong>{{ .FormattedDue }}</strong>.
    </div>
    <div class="footer-right">
      <div class="footer-brand">
        generated with <span>tb</span> — time bill
      </div>
    </div>
  </div>

</div>
</body>
</html>`
