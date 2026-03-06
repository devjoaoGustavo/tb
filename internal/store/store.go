package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/joaogustavo/tb/internal/model"
	_ "modernc.org/sqlite"
)

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

// Open creates or opens the database at the given path.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_journal=WAL&_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("opening db: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("pinging db: %w", err)
	}
	if err := migrate(db); err != nil {
		return nil, fmt.Errorf("migrating db: %w", err)
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS clients (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT,
    company TEXT,
    address TEXT,
    tax_id TEXT,
    phone TEXT,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    client_id TEXT NOT NULL REFERENCES clients(id),
    billing_type TEXT NOT NULL,
    hourly_rate REAL,
    fixed_amount REAL,
    currency TEXT NOT NULL,
    active INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id),
    start_at DATETIME NOT NULL,
    end_at DATETIME,
    note TEXT,
    tags TEXT,
    billed INTEGER NOT NULL DEFAULT 0
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_active_session ON sessions(project_id) WHERE end_at IS NULL;

CREATE TABLE IF NOT EXISTS invoices (
    id TEXT PRIMARY KEY,
    number TEXT NOT NULL UNIQUE,
    client_id TEXT NOT NULL,
    project_id TEXT NOT NULL,
    status TEXT NOT NULL,
    billing_type TEXT NOT NULL,
    currency TEXT NOT NULL,
    line_items TEXT NOT NULL,
    subtotal REAL NOT NULL,
    tax REAL,
    tax_rate REAL,
    total REAL NOT NULL,
    issued_at DATETIME,
    due_at DATETIME,
    paid_at DATETIME,
    notes TEXT,
    period_start DATETIME,
    period_end DATETIME,
    created_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS sequences (
    client_id TEXT PRIMARY KEY,
    next_number INTEGER NOT NULL DEFAULT 1
);
`)
	return err
}

// --- Clients ---

// CreateClient inserts a new client record.
func (s *Store) CreateClient(c model.Client) error {
	_, err := s.db.Exec(
		`INSERT INTO clients (id, name, email, company, address, tax_id, phone, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.Email, c.Company, c.Address, c.TaxID, c.Phone, c.Created.UTC(),
	)
	return err
}

// ListClients returns all clients.
func (s *Store) ListClients() ([]model.Client, error) {
	rows, err := s.db.Query(
		`SELECT id, name, email, company, address, tax_id, phone, created_at FROM clients ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clients []model.Client
	for rows.Next() {
		var c model.Client
		var createdAt string
		if err := rows.Scan(&c.ID, &c.Name, &c.Email, &c.Company, &c.Address, &c.TaxID, &c.Phone, &createdAt); err != nil {
			return nil, err
		}
		c.Created, _ = time.Parse(time.RFC3339, createdAt)
		clients = append(clients, c)
	}
	return clients, rows.Err()
}

// GetClientByID returns a client by its slug ID.
func (s *Store) GetClientByID(id string) (model.Client, error) {
	var c model.Client
	var createdAt string
	err := s.db.QueryRow(
		`SELECT id, name, email, company, address, tax_id, phone, created_at FROM clients WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.Email, &c.Company, &c.Address, &c.TaxID, &c.Phone, &createdAt)
	if err == sql.ErrNoRows {
		return c, fmt.Errorf("client %q not found", id)
	}
	if err != nil {
		return c, err
	}
	c.Created, _ = time.Parse(time.RFC3339, createdAt)
	return c, nil
}

// DeleteClient removes a client and all their projects, sessions, invoices, and sequences.
func (s *Store) DeleteClient(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint

	rows, err := tx.Query(`SELECT id FROM projects WHERE client_id = ?`, id)
	if err != nil {
		return err
	}
	var projectIDs []string
	for rows.Next() {
		var pid string
		if err := rows.Scan(&pid); err != nil {
			rows.Close()
			return err
		}
		projectIDs = append(projectIDs, pid)
	}
	rows.Close()

	for _, pid := range projectIDs {
		if _, err := tx.Exec(`DELETE FROM sessions WHERE project_id = ?`, pid); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`DELETE FROM projects WHERE client_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM invoices WHERE client_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM sequences WHERE client_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM clients WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Projects ---

// CreateProject inserts a new project record.
func (s *Store) CreateProject(p model.Project) error {
	active := 0
	if p.Active {
		active = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO projects (id, name, client_id, billing_type, hourly_rate, fixed_amount, currency, active, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.Name, p.ClientID, string(p.BillingType), p.HourlyRate, p.FixedAmount, string(p.Currency), active, p.Created.UTC(),
	)
	return err
}

// ListProjects returns all projects, optionally filtered by clientID.
func (s *Store) ListProjects(clientID string) ([]model.Project, error) {
	query := `SELECT id, name, client_id, billing_type, hourly_rate, fixed_amount, currency, active, created_at FROM projects`
	args := []any{}
	if clientID != "" {
		query += ` WHERE client_id = ?`
		args = append(args, clientID)
	}
	query += ` ORDER BY name`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []model.Project
	for rows.Next() {
		var p model.Project
		var active int
		var createdAt string
		if err := rows.Scan(&p.ID, &p.Name, &p.ClientID, &p.BillingType, &p.HourlyRate, &p.FixedAmount, &p.Currency, &active, &createdAt); err != nil {
			return nil, err
		}
		p.Active = active == 1
		p.Created, _ = time.Parse(time.RFC3339, createdAt)
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

// GetProjectByID returns a project by its slug ID.
func (s *Store) GetProjectByID(id string) (model.Project, error) {
	var p model.Project
	var active int
	var createdAt string
	err := s.db.QueryRow(
		`SELECT id, name, client_id, billing_type, hourly_rate, fixed_amount, currency, active, created_at FROM projects WHERE id = ?`, id,
	).Scan(&p.ID, &p.Name, &p.ClientID, &p.BillingType, &p.HourlyRate, &p.FixedAmount, &p.Currency, &active, &createdAt)
	if err == sql.ErrNoRows {
		return p, fmt.Errorf("project %q not found", id)
	}
	if err != nil {
		return p, err
	}
	p.Active = active == 1
	p.Created, _ = time.Parse(time.RFC3339, createdAt)
	return p, nil
}

// DeleteProject removes a project and all its sessions.
func (s *Store) DeleteProject(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint

	if _, err := tx.Exec(`DELETE FROM sessions WHERE project_id = ?`, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM projects WHERE id = ?`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// --- Sessions ---

func sessionID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func tagsToString(tags []string) string {
	return strings.Join(tags, ",")
}

func tagsFromString(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func nullableTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC().Format(time.RFC3339)
}

func scanTime(s *string) time.Time {
	if s == nil || *s == "" {
		return time.Time{}
	}
	t, _ := time.Parse(time.RFC3339, *s)
	return t
}

// CreateSession inserts a new session record.
func (s *Store) CreateSession(sess model.Session) error {
	if sess.ID == "" {
		sess.ID = sessionID()
	}
	billed := 0
	if sess.Billed {
		billed = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO sessions (id, project_id, start_at, end_at, note, tags, billed)
         VALUES (?, ?, ?, ?, ?, ?, ?)`,
		sess.ID, sess.ProjectID, sess.Start.UTC().Format(time.RFC3339),
		nullableTime(sess.End), sess.Note, tagsToString(sess.Tags), billed,
	)
	return err
}

// ActiveSession returns the currently running session, or nil if none.
func (s *Store) ActiveSession() (*model.Session, error) {
	var sess model.Session
	var endAt *string
	var startAt string
	var tags string
	var billed int
	err := s.db.QueryRow(
		`SELECT id, project_id, start_at, end_at, note, tags, billed FROM sessions WHERE end_at IS NULL LIMIT 1`,
	).Scan(&sess.ID, &sess.ProjectID, &startAt, &endAt, &sess.Note, &tags, &billed)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	sess.Start, _ = time.Parse(time.RFC3339, startAt)
	sess.Tags = tagsFromString(tags)
	sess.Billed = billed == 1
	if endAt != nil {
		sess.End, _ = time.Parse(time.RFC3339, *endAt)
	}
	return &sess, nil
}

// UpdateSession updates the end_at and note of an existing session.
func (s *Store) UpdateSession(sess model.Session) error {
	_, err := s.db.Exec(
		`UPDATE sessions SET end_at = ?, note = ? WHERE id = ?`,
		nullableTime(sess.End), sess.Note, sess.ID,
	)
	return err
}

// DeleteSession removes a session by ID.
func (s *Store) DeleteSession(id string) error {
	_, err := s.db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// ListSessions returns sessions for a project (or all if projectID is "") in [from, to].
func (s *Store) ListSessions(projectID string, from, to time.Time) ([]model.Session, error) {
	query := `SELECT id, project_id, start_at, end_at, note, tags, billed FROM sessions WHERE 1=1`
	args := []any{}
	if projectID != "" {
		query += ` AND project_id = ?`
		args = append(args, projectID)
	}
	if !from.IsZero() {
		query += ` AND start_at >= ?`
		args = append(args, from.UTC().Format(time.RFC3339))
	}
	if !to.IsZero() {
		query += ` AND start_at <= ?`
		args = append(args, to.UTC().Format(time.RFC3339))
	}
	query += ` ORDER BY start_at`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// ListSessionsByClient returns sessions for all projects belonging to a client.
func (s *Store) ListSessionsByClient(clientID string, from, to time.Time) ([]model.Session, error) {
	query := `SELECT s.id, s.project_id, s.start_at, s.end_at, s.note, s.tags, s.billed
              FROM sessions s
              JOIN projects p ON p.id = s.project_id
              WHERE p.client_id = ?`
	args := []any{clientID}
	if !from.IsZero() {
		query += ` AND s.start_at >= ?`
		args = append(args, from.UTC().Format(time.RFC3339))
	}
	if !to.IsZero() {
		query += ` AND s.start_at <= ?`
		args = append(args, to.UTC().Format(time.RFC3339))
	}
	query += ` ORDER BY s.start_at`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSessions(rows)
}

// UnmarkSessionsBilled resets billed=0 for the given session IDs.
func (s *Store) UnmarkSessionsBilled(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	_, err := s.db.Exec(`UPDATE sessions SET billed = 0 WHERE id IN (`+placeholders+`)`, args...)
	return err
}

// MarkSessionsBilled marks the given session IDs as billed.
func (s *Store) MarkSessionsBilled(ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	_, err := s.db.Exec(`UPDATE sessions SET billed = 1 WHERE id IN (`+placeholders+`)`, args...)
	return err
}

func scanSessions(rows *sql.Rows) ([]model.Session, error) {
	var sessions []model.Session
	for rows.Next() {
		var sess model.Session
		var endAt *string
		var startAt string
		var tags string
		var billed int
		if err := rows.Scan(&sess.ID, &sess.ProjectID, &startAt, &endAt, &sess.Note, &tags, &billed); err != nil {
			return nil, err
		}
		sess.Start, _ = time.Parse(time.RFC3339, startAt)
		sess.Tags = tagsFromString(tags)
		sess.Billed = billed == 1
		if endAt != nil {
			sess.End, _ = time.Parse(time.RFC3339, *endAt)
		}
		sessions = append(sessions, sess)
	}
	return sessions, rows.Err()
}

// --- Invoices ---

// CreateInvoice inserts a new invoice record, marshalling LineItems to JSON.
func (s *Store) CreateInvoice(inv model.Invoice) error {
	lineItems, err := json.Marshal(inv.LineItems)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(
		`INSERT INTO invoices (id, number, client_id, project_id, status, billing_type, currency,
         line_items, subtotal, tax, tax_rate, total, issued_at, due_at, paid_at, notes,
         period_start, period_end, created_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.Number, inv.ClientID, inv.ProjectID, string(inv.Status),
		string(inv.BillingType), string(inv.Currency), string(lineItems),
		inv.Subtotal, inv.Tax, inv.TaxRate, inv.Total,
		nullableTime(inv.IssuedAt), nullableTime(inv.DueAt), nullableTime(inv.PaidAt),
		inv.Notes, nullableTime(inv.PeriodStart), nullableTime(inv.PeriodEnd),
		inv.Created.UTC().Format(time.RFC3339),
	)
	return err
}

// ListInvoices returns invoices, optionally filtered by status.
func (s *Store) ListInvoices(status string) ([]model.Invoice, error) {
	query := `SELECT id, number, client_id, project_id, status, billing_type, currency,
              line_items, subtotal, tax, tax_rate, total, issued_at, due_at, paid_at, notes,
              period_start, period_end, created_at FROM invoices`
	args := []any{}
	if status != "" {
		query += ` WHERE status = ?`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInvoices(rows)
}

// GetInvoiceByNumber returns an invoice by its display number.
func (s *Store) GetInvoiceByNumber(number string) (model.Invoice, error) {
	rows, err := s.db.Query(
		`SELECT id, number, client_id, project_id, status, billing_type, currency,
         line_items, subtotal, tax, tax_rate, total, issued_at, due_at, paid_at, notes,
         period_start, period_end, created_at FROM invoices WHERE number = ?`, number,
	)
	if err != nil {
		return model.Invoice{}, err
	}
	defer rows.Close()
	invs, err := scanInvoices(rows)
	if err != nil {
		return model.Invoice{}, err
	}
	if len(invs) == 0 {
		return model.Invoice{}, fmt.Errorf("invoice %q not found", number)
	}
	return invs[0], nil
}

// UpdateInvoiceStatus updates the status (and optionally paidAt) of an invoice.
func (s *Store) UpdateInvoiceStatus(number string, status model.InvoiceStatus, paidAt time.Time) error {
	_, err := s.db.Exec(
		`UPDATE invoices SET status = ?, paid_at = ? WHERE number = ?`,
		string(status), nullableTime(paidAt), number,
	)
	return err
}

// DeleteInvoice removes an invoice by its display number.
func (s *Store) DeleteInvoice(number string) error {
	_, err := s.db.Exec(`DELETE FROM invoices WHERE number = ?`, number)
	return err
}

func scanInvoices(rows *sql.Rows) ([]model.Invoice, error) {
	var invoices []model.Invoice
	for rows.Next() {
		var inv model.Invoice
		var lineItemsJSON string
		var issuedAt, dueAt, paidAt, periodStart, periodEnd *string
		var createdAt string
		if err := rows.Scan(
			&inv.ID, &inv.Number, &inv.ClientID, &inv.ProjectID,
			&inv.Status, &inv.BillingType, &inv.Currency,
			&lineItemsJSON, &inv.Subtotal, &inv.Tax, &inv.TaxRate, &inv.Total,
			&issuedAt, &dueAt, &paidAt, &inv.Notes,
			&periodStart, &periodEnd, &createdAt,
		); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(lineItemsJSON), &inv.LineItems); err != nil {
			return nil, err
		}
		inv.Created, _ = time.Parse(time.RFC3339, createdAt)
		if issuedAt != nil {
			inv.IssuedAt, _ = time.Parse(time.RFC3339, *issuedAt)
		}
		if dueAt != nil {
			inv.DueAt, _ = time.Parse(time.RFC3339, *dueAt)
		}
		if paidAt != nil {
			inv.PaidAt, _ = time.Parse(time.RFC3339, *paidAt)
		}
		if periodStart != nil {
			inv.PeriodStart, _ = time.Parse(time.RFC3339, *periodStart)
		}
		if periodEnd != nil {
			inv.PeriodEnd, _ = time.Parse(time.RFC3339, *periodEnd)
		}
		invoices = append(invoices, inv)
	}
	return invoices, rows.Err()
}

// --- Sequences ---

// NextClientSequence atomically returns the current sequence number for a client
// and increments it for the next call. Returns the number before increment.
func (s *Store) NextClientSequence(clientID string) (int, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback() //nolint

	_, err = tx.Exec(`INSERT OR IGNORE INTO sequences (client_id, next_number) VALUES (?, 1)`, clientID)
	if err != nil {
		return 0, err
	}

	var num int
	if err := tx.QueryRow(`SELECT next_number FROM sequences WHERE client_id = ?`, clientID).Scan(&num); err != nil {
		return 0, err
	}

	_, err = tx.Exec(`UPDATE sequences SET next_number = next_number + 1 WHERE client_id = ?`, clientID)
	if err != nil {
		return 0, err
	}

	return num, tx.Commit()
}

// PeekClientSequence reads the current sequence number without advancing it.
func (s *Store) PeekClientSequence(clientID string) (int, error) {
	var num int
	err := s.db.QueryRow(`SELECT next_number FROM sequences WHERE client_id = ?`, clientID).Scan(&num)
	if err == sql.ErrNoRows {
		return 1, nil
	}
	return num, err
}
