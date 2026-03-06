package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/devjoaoGustavo/tb/internal/model"
)

// --- helpers ---

func openTestDB(t *testing.T) *Store {
	t.Helper()
	st, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func makeClient(id, name string) model.Client {
	return model.Client{ID: id, Name: name, Email: id + "@example.com", Created: time.Now()}
}

func makeProject(id, clientID string) model.Project {
	return model.Project{
		ID: id, Name: id, ClientID: clientID,
		BillingType: model.BillingHourly, HourlyRate: 100,
		Currency: model.CurrencyUSD, Active: true, Created: time.Now(),
	}
}

func makeSession(id, projectID string, start, end time.Time) model.Session {
	return model.Session{ID: id, ProjectID: projectID, Start: start, End: end}
}

func makeInvoice(id, number, clientID, projectID string) model.Invoice {
	return model.Invoice{
		ID: id, Number: number,
		ClientID: clientID, ProjectID: projectID,
		Status: model.InvoiceDraft, BillingType: model.BillingHourly,
		Currency:  model.CurrencyUSD,
		LineItems: []model.LineItem{{Description: "work", Hours: 1, Rate: 100, Amount: 100}},
		Subtotal:  100, Total: 100,
		IssuedAt:  time.Now(), DueAt: time.Now().AddDate(0, 0, 15),
		Created:   time.Now(),
	}
}

// --- clients ---

func TestCreateClient_GetClientByID(t *testing.T) {
	st := openTestDB(t)

	c := makeClient("acme", "ACME Corp")
	if err := st.CreateClient(c); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetClientByID("acme")
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "ACME Corp" {
		t.Errorf("Name = %q, want ACME Corp", got.Name)
	}
	if got.Email != "acme@example.com" {
		t.Errorf("Email = %q, want acme@example.com", got.Email)
	}
}

func TestGetClientByID_NotFound(t *testing.T) {
	st := openTestDB(t)
	if _, err := st.GetClientByID("ghost"); err == nil {
		t.Error("expected error for missing client")
	}
}

func TestListClients_Empty(t *testing.T) {
	st := openTestDB(t)
	clients, err := st.ListClients()
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 0 {
		t.Errorf("want 0, got %d", len(clients))
	}
}

func TestListClients_Multiple(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("a", "Alpha"))
	st.CreateClient(makeClient("b", "Beta"))

	clients, err := st.ListClients()
	if err != nil {
		t.Fatal(err)
	}
	if len(clients) != 2 {
		t.Errorf("want 2, got %d", len(clients))
	}
}

func TestDeleteClient_Cascade(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))
	st.CreateSession(makeSession("s1", "p", time.Now().Add(-time.Hour), time.Now()))
	st.CreateInvoice(makeInvoice("i1", "INV-001", "c", "p"))

	if err := st.DeleteClient("c"); err != nil {
		t.Fatal(err)
	}

	if _, err := st.GetClientByID("c"); err == nil {
		t.Error("client should be deleted")
	}
	if _, err := st.GetProjectByID("p"); err == nil {
		t.Error("project should be cascade-deleted")
	}
	sessions, _ := st.ListSessions("p", time.Time{}, time.Time{})
	if len(sessions) != 0 {
		t.Errorf("sessions should be cascade-deleted, got %d", len(sessions))
	}
	if _, err := st.GetInvoiceByNumber("INV-001"); err == nil {
		t.Error("invoice should be cascade-deleted")
	}
}

// --- projects ---

func TestCreateProject_GetProjectByID(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))

	p := makeProject("proj", "c")
	p.HourlyRate = 150
	if err := st.CreateProject(p); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetProjectByID("proj")
	if err != nil {
		t.Fatal(err)
	}
	if got.ClientID != "c" {
		t.Errorf("ClientID = %q, want c", got.ClientID)
	}
	if got.HourlyRate != 150 {
		t.Errorf("HourlyRate = %f, want 150", got.HourlyRate)
	}
	if !got.Active {
		t.Error("Active should be true")
	}
}

func TestGetProjectByID_NotFound(t *testing.T) {
	st := openTestDB(t)
	if _, err := st.GetProjectByID("ghost"); err == nil {
		t.Error("expected error for missing project")
	}
}

func TestListProjects_All(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c1", "C1"))
	st.CreateClient(makeClient("c2", "C2"))
	st.CreateProject(makeProject("p1", "c1"))
	st.CreateProject(makeProject("p2", "c2"))

	all, err := st.ListProjects("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Errorf("want 2 total, got %d", len(all))
	}
}

func TestListProjects_FilterByClient(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c1", "C1"))
	st.CreateClient(makeClient("c2", "C2"))
	st.CreateProject(makeProject("p1", "c1"))
	st.CreateProject(makeProject("p2", "c2"))

	filtered, err := st.ListProjects("c1")
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 1 || filtered[0].ID != "p1" {
		t.Errorf("want [p1], got %v", filtered)
	}
}

func TestDeleteProject_CascadesSessions(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))
	st.CreateSession(makeSession("s1", "p", time.Now().Add(-time.Hour), time.Now()))

	if err := st.DeleteProject("p"); err != nil {
		t.Fatal(err)
	}

	if _, err := st.GetProjectByID("p"); err == nil {
		t.Error("project should be deleted")
	}
	sessions, _ := st.ListSessions("p", time.Time{}, time.Time{})
	if len(sessions) != 0 {
		t.Errorf("sessions should be deleted with project, got %d", len(sessions))
	}
}

// --- sessions ---

func TestActiveSession_NoneRunning(t *testing.T) {
	st := openTestDB(t)
	active, err := st.ActiveSession()
	if err != nil {
		t.Fatal(err)
	}
	if active != nil {
		t.Errorf("expected nil, got %+v", active)
	}
}

func TestCreateSession_ActiveSession(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	sess := model.Session{ID: "s1", ProjectID: "p", Start: time.Now()}
	if err := st.CreateSession(sess); err != nil {
		t.Fatal(err)
	}

	active, err := st.ActiveSession()
	if err != nil {
		t.Fatal(err)
	}
	if active == nil || active.ID != "s1" {
		t.Errorf("expected active session s1, got %v", active)
	}
}

func TestActiveSession_UniqueConstraint(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	st.CreateSession(model.Session{ID: "s1", ProjectID: "p", Start: time.Now()})
	err := st.CreateSession(model.Session{ID: "s2", ProjectID: "p", Start: time.Now()})
	if err == nil {
		t.Error("expected error: only one active session per project allowed")
	}
}

func TestUpdateSession_StopsTimer(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	start := time.Now().Add(-time.Hour)
	st.CreateSession(model.Session{ID: "s1", ProjectID: "p", Start: start})

	end := time.Now()
	if err := st.UpdateSession(model.Session{ID: "s1", ProjectID: "p", Start: start, End: end, Note: "done"}); err != nil {
		t.Fatal(err)
	}

	active, _ := st.ActiveSession()
	if active != nil {
		t.Error("session should no longer be active after update")
	}
}

func TestDeleteSession(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))
	st.CreateSession(model.Session{ID: "s1", ProjectID: "p", Start: time.Now()})

	if err := st.DeleteSession("s1"); err != nil {
		t.Fatal(err)
	}

	active, _ := st.ActiveSession()
	if active != nil {
		t.Error("session should be gone after delete")
	}
}

func TestListSessions_AllForProject(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	now := time.Now()
	st.CreateSession(makeSession("s1", "p", now.Add(-2*time.Hour), now.Add(-time.Hour)))
	st.CreateSession(makeSession("s2", "p", now.Add(-time.Hour), now))

	sessions, err := st.ListSessions("p", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Errorf("want 2, got %d", len(sessions))
	}
}

func TestListSessions_DateFilter(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	old := time.Now().Add(-48 * time.Hour)
	recent := time.Now().Add(-time.Hour)
	st.CreateSession(makeSession("s1", "p", old, old.Add(time.Hour)))
	st.CreateSession(makeSession("s2", "p", recent, recent.Add(30*time.Minute)))

	from := time.Now().Add(-24 * time.Hour)
	sessions, err := st.ListSessions("p", from, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "s2" {
		t.Errorf("expected [s2], got %v", sessions)
	}
}

func TestListSessions_AllProjects(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p1", "c"))
	st.CreateProject(makeProject("p2", "c"))

	now := time.Now()
	st.CreateSession(makeSession("s1", "p1", now.Add(-time.Hour), now))
	st.CreateSession(makeSession("s2", "p2", now.Add(-time.Hour), now))

	sessions, err := st.ListSessions("", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Errorf("want 2, got %d", len(sessions))
	}
}

func TestListSessionsByClient(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c1", "C1"))
	st.CreateClient(makeClient("c2", "C2"))
	st.CreateProject(makeProject("p1", "c1"))
	st.CreateProject(makeProject("p2", "c2"))

	now := time.Now()
	st.CreateSession(makeSession("s1", "p1", now.Add(-time.Hour), now))
	st.CreateSession(makeSession("s2", "p2", now.Add(-time.Hour), now))

	sessions, err := st.ListSessionsByClient("c1", time.Time{}, time.Time{})
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 || sessions[0].ID != "s1" {
		t.Errorf("expected [s1], got %v", sessions)
	}
}

func TestMarkSessionsBilled(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	now := time.Now()
	st.CreateSession(makeSession("s1", "p", now.Add(-2*time.Hour), now.Add(-time.Hour)))
	st.CreateSession(makeSession("s2", "p", now.Add(-time.Hour), now))

	if err := st.MarkSessionsBilled([]string{"s1"}); err != nil {
		t.Fatal(err)
	}

	sessions, _ := st.ListSessions("p", time.Time{}, time.Time{})
	billed := 0
	for _, s := range sessions {
		if s.Billed {
			billed++
		}
	}
	if billed != 1 {
		t.Errorf("want 1 billed session, got %d", billed)
	}
}

func TestMarkSessionsBilled_Empty(t *testing.T) {
	st := openTestDB(t)
	if err := st.MarkSessionsBilled([]string{}); err != nil {
		t.Errorf("MarkSessionsBilled with empty slice should not error: %v", err)
	}
}

func TestSession_Tags_RoundTrip(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	sess := model.Session{
		ID:        "s1",
		ProjectID: "p",
		Start:     time.Now().Add(-time.Hour),
		End:       time.Now(),
		Tags:      []string{"backend", "review"},
	}
	if err := st.CreateSession(sess); err != nil {
		t.Fatal(err)
	}

	sessions, _ := st.ListSessions("p", time.Time{}, time.Time{})
	if len(sessions) != 1 {
		t.Fatalf("want 1 session, got %d", len(sessions))
	}
	if len(sessions[0].Tags) != 2 || sessions[0].Tags[0] != "backend" {
		t.Errorf("tags round-trip failed: %v", sessions[0].Tags)
	}
}

// --- invoices ---

func TestCreateInvoice_GetInvoiceByNumber(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	inv := makeInvoice("id1", "INV-001", "c", "p")
	inv.TaxRate = 0.1
	inv.Tax = 10
	inv.Total = 110
	inv.Notes = "payment in 15 days"
	if err := st.CreateInvoice(inv); err != nil {
		t.Fatal(err)
	}

	got, err := st.GetInvoiceByNumber("INV-001")
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 110 {
		t.Errorf("Total = %f, want 110", got.Total)
	}
	if got.TaxRate != 0.1 {
		t.Errorf("TaxRate = %f, want 0.1", got.TaxRate)
	}
	if got.Notes != "payment in 15 days" {
		t.Errorf("Notes = %q, want 'payment in 15 days'", got.Notes)
	}
	if len(got.LineItems) != 1 || got.LineItems[0].Description != "work" {
		t.Errorf("LineItems not round-tripped: %v", got.LineItems)
	}
}

func TestGetInvoiceByNumber_NotFound(t *testing.T) {
	st := openTestDB(t)
	if _, err := st.GetInvoiceByNumber("INV-999"); err == nil {
		t.Error("expected error for missing invoice")
	}
}

func TestListInvoices_NoFilter(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))
	st.CreateInvoice(makeInvoice("i1", "INV-001", "c", "p"))
	st.CreateInvoice(makeInvoice("i2", "INV-002", "c", "p"))

	invoices, err := st.ListInvoices("")
	if err != nil {
		t.Fatal(err)
	}
	if len(invoices) != 2 {
		t.Errorf("want 2, got %d", len(invoices))
	}
}

func TestListInvoices_StatusFilter(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))

	inv1 := makeInvoice("i1", "INV-001", "c", "p")
	inv2 := makeInvoice("i2", "INV-002", "c", "p")
	inv2.Status = model.InvoicePaid
	st.CreateInvoice(inv1)
	st.CreateInvoice(inv2)

	paid, err := st.ListInvoices("paid")
	if err != nil {
		t.Fatal(err)
	}
	if len(paid) != 1 || paid[0].Number != "INV-002" {
		t.Errorf("want [INV-002], got %v", paid)
	}

	draft, _ := st.ListInvoices("draft")
	if len(draft) != 1 || draft[0].Number != "INV-001" {
		t.Errorf("want [INV-001], got %v", draft)
	}
}

func TestUpdateInvoiceStatus(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))
	st.CreateInvoice(makeInvoice("i1", "INV-001", "c", "p"))

	paidAt := time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC)
	if err := st.UpdateInvoiceStatus("INV-001", model.InvoicePaid, paidAt); err != nil {
		t.Fatal(err)
	}

	got, _ := st.GetInvoiceByNumber("INV-001")
	if got.Status != model.InvoicePaid {
		t.Errorf("Status = %q, want paid", got.Status)
	}
	if !got.PaidAt.Equal(paidAt) {
		t.Errorf("PaidAt = %v, want %v", got.PaidAt, paidAt)
	}
}

func TestUpdateInvoiceStatus_Sent(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))
	st.CreateInvoice(makeInvoice("i1", "INV-001", "c", "p"))

	if err := st.UpdateInvoiceStatus("INV-001", model.InvoiceSent, time.Time{}); err != nil {
		t.Fatal(err)
	}

	got, _ := st.GetInvoiceByNumber("INV-001")
	if got.Status != model.InvoiceSent {
		t.Errorf("Status = %q, want sent", got.Status)
	}
	if !got.PaidAt.IsZero() {
		t.Error("PaidAt should be zero for sent status")
	}
}

func TestDeleteInvoice(t *testing.T) {
	st := openTestDB(t)
	st.CreateClient(makeClient("c", "C"))
	st.CreateProject(makeProject("p", "c"))
	st.CreateInvoice(makeInvoice("i1", "INV-001", "c", "p"))

	if err := st.DeleteInvoice("INV-001"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.GetInvoiceByNumber("INV-001"); err == nil {
		t.Error("invoice should be deleted")
	}
}

// --- sequences ---

func TestNextClientSequence_StartsAtOne(t *testing.T) {
	st := openTestDB(t)

	n, err := st.NextClientSequence("c")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("first call = %d, want 1", n)
	}
}

func TestNextClientSequence_Increments(t *testing.T) {
	st := openTestDB(t)

	n1, _ := st.NextClientSequence("c")
	n2, _ := st.NextClientSequence("c")
	n3, _ := st.NextClientSequence("c")

	if n1 != 1 || n2 != 2 || n3 != 3 {
		t.Errorf("sequence = %d, %d, %d; want 1, 2, 3", n1, n2, n3)
	}
}

func TestNextClientSequence_IndependentPerClient(t *testing.T) {
	st := openTestDB(t)

	st.NextClientSequence("c1")
	st.NextClientSequence("c1")

	n, err := st.NextClientSequence("c2")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("c2 sequence = %d, want 1", n)
	}
}

func TestPeekClientSequence_DoesNotAdvance(t *testing.T) {
	st := openTestDB(t)

	st.NextClientSequence("c")
	st.NextClientSequence("c")

	n1, err := st.PeekClientSequence("c")
	if err != nil {
		t.Fatal(err)
	}
	n2, _ := st.PeekClientSequence("c")

	if n1 != 3 {
		t.Errorf("peek after 2 advances = %d, want 3", n1)
	}
	if n1 != n2 {
		t.Errorf("peek should be idempotent: %d != %d", n1, n2)
	}
}

func TestPeekClientSequence_BeforeAnyAdvance(t *testing.T) {
	st := openTestDB(t)

	n, err := st.PeekClientSequence("new-client")
	if err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Errorf("peek before any advance = %d, want 1", n)
	}
}
