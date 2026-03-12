package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// integrationBinary is built once by TestMain and shared across all tests.
var integrationBinary string

func TestMain(m *testing.M) {
	bin, err := os.MkdirTemp("", "tb-integration-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "MkdirTemp: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(bin)

	integrationBinary = filepath.Join(bin, "tb")
	build := exec.Command("go", "build", "-o", integrationBinary, ".")
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

// testEnv holds isolated config/data directories for one test.
type testEnv struct {
	t         *testing.T
	configDir string
	dataDir   string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	configDir := t.TempDir()
	dataDir := t.TempDir()

	// Write a minimal config so commands don't print "Created default config" noise.
	cfgJSON := `{
  "issuer": {"name": "Test User", "email": "test@example.com"},
  "invoice": {
    "prefix": "INV",
    "start_number": 1,
    "padding": 3,
    "separator": "-",
    "include_year": false,
    "default_due_days": 15,
    "default_currency": "USD"
  }
}`
	if err := os.WriteFile(filepath.Join(configDir, "config.json"), []byte(cfgJSON), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return &testEnv{t: t, configDir: configDir, dataDir: dataDir}
}

// run executes the binary with the test environment and returns stdout, stderr, exit code.
func (e *testEnv) run(args ...string) (string, string, int) {
	cmd := exec.Command(integrationBinary, args...)
	cmd.Env = append(os.Environ(),
		"TB_CONFIG_DIR="+e.configDir,
		"TB_DATA_DIR="+e.dataDir,
		"EDITOR=true", // prevent blocking on $EDITOR
	)
	var out, errBuf bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &errBuf
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	}
	return out.String(), errBuf.String(), code
}

// mustRun asserts exit code 0 and returns stdout.
func (e *testEnv) mustRun(args ...string) string {
	e.t.Helper()
	stdout, stderr, code := e.run(args...)
	if code != 0 {
		e.t.Fatalf("tb %v exited %d\nstderr: %s\nstdout: %s", args, code, stderr, stdout)
	}
	return stdout
}

// mustFail asserts a non-zero exit code.
func (e *testEnv) mustFail(args ...string) (string, string) {
	e.t.Helper()
	stdout, stderr, code := e.run(args...)
	if code == 0 {
		e.t.Fatalf("tb %v expected failure but exited 0\nstdout: %s", args, stdout)
	}
	return stdout, stderr
}

func contains(s, sub string) bool { return strings.Contains(s, sub) }

// --- Helpers to set up common entities ---

func (e *testEnv) addClient(name string) {
	e.t.Helper()
	e.mustRun("client", "add", name, "--email", "test@example.com")
}

func (e *testEnv) addProject(name, clientID string) {
	e.t.Helper()
	e.mustRun("project", "add", name, "--client", clientID, "--rate", "100", "--currency", "USD")
}

func (e *testEnv) logTime(projectID, duration, note string) {
	e.t.Helper()
	e.mustRun("log", projectID, duration, note)
}

// --- Tests ---

func TestIntegration_ClientCRUD(t *testing.T) {
	env := newTestEnv(t)

	// Add
	out := env.mustRun("client", "add", "ACME Corp", "--email", "cfo@acme.com", "--company", "ACME Inc")
	if !contains(out, "acme-corp") {
		t.Errorf("add: expected slug in output, got: %q", out)
	}

	// List
	out = env.mustRun("client", "list")
	if !contains(out, "acme-corp") || !contains(out, "ACME Corp") {
		t.Errorf("list: expected client, got: %q", out)
	}

	// Delete with --force
	out = env.mustRun("client", "delete", "acme-corp", "--force")
	if !contains(out, "deleted") {
		t.Errorf("delete: expected 'deleted', got: %q", out)
	}

	// Confirm gone
	out = env.mustRun("client", "list")
	if contains(out, "acme-corp") {
		t.Errorf("after delete: client still appears in list: %q", out)
	}
}

func TestIntegration_ClientDelete_NotFound(t *testing.T) {
	env := newTestEnv(t)
	_, stderr := env.mustFail("client", "delete", "no-such-client", "--force")
	if !contains(stderr, "not found") && !contains(stderr, "no such") && !contains(stderr, "no-such-client") {
		t.Errorf("expected not-found error, got: %q", stderr)
	}
}

func TestIntegration_ProjectCRUD(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")

	// Add
	out := env.mustRun("project", "add", "API Work", "--client", "acme-corp", "--rate", "150", "--currency", "USD")
	if !contains(out, "api-work") {
		t.Errorf("add: expected slug in output, got: %q", out)
	}

	// List all
	out = env.mustRun("project", "list")
	if !contains(out, "api-work") {
		t.Errorf("list: expected project, got: %q", out)
	}

	// List filtered by client
	out = env.mustRun("project", "list", "--client", "acme-corp")
	if !contains(out, "api-work") {
		t.Errorf("list --client: expected project, got: %q", out)
	}

	// Delete
	out = env.mustRun("project", "delete", "api-work", "--force")
	if !contains(out, "deleted") {
		t.Errorf("delete: expected 'deleted', got: %q", out)
	}

	out = env.mustRun("project", "list")
	if contains(out, "api-work") {
		t.Errorf("after delete: project still in list: %q", out)
	}
}

func TestIntegration_TimeTracking(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")

	// Start
	out := env.mustRun("start", "api-work")
	if !contains(out, "api-work") {
		t.Errorf("start: expected project in output, got: %q", out)
	}

	// Now (timer running)
	out = env.mustRun("now")
	if !contains(out, "api-work") {
		t.Errorf("now: expected running timer, got: %q", out)
	}

	// Can't start another
	_, stderr, code := env.run("start", "api-work")
	if code == 0 {
		t.Error("start while running: expected failure")
	}
	if !contains(stderr, "already running") {
		t.Errorf("start while running: unexpected error: %q", stderr)
	}

	// Stop
	out = env.mustRun("stop", "--note", "auth module")
	if !contains(out, "api-work") {
		t.Errorf("stop: expected project in output, got: %q", out)
	}

	// Now (idle)
	out = env.mustRun("now")
	if !contains(out, "No timer running") {
		t.Errorf("now after stop: expected idle, got: %q", out)
	}
}

func TestIntegration_StartBackdated(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")

	env.mustRun("start", "api-work", "--at", "30m ago")
	out := env.mustRun("stop")
	if !contains(out, "api-work") {
		t.Errorf("unexpected stop output: %q", out)
	}
}

func TestIntegration_Cancel(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")

	env.mustRun("start", "api-work")
	out := env.mustRun("cancel")
	if !contains(out, "Cancelled") {
		t.Errorf("cancel: expected 'Cancelled', got: %q", out)
	}

	out = env.mustRun("now")
	if !contains(out, "No timer running") {
		t.Errorf("after cancel: expected idle, got: %q", out)
	}
}

func TestIntegration_Switch(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")
	env.addProject("Design Work", "acme-corp")

	env.mustRun("start", "api-work")
	env.mustRun("switch", "design-work")

	out := env.mustRun("now")
	if !contains(out, "design-work") {
		t.Errorf("switch: expected design-work running, got: %q", out)
	}
	env.mustRun("stop")
}

func TestIntegration_Log(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")

	out := env.mustRun("log", "api-work", "2h30m", "Sprint work")
	if !contains(out, "2.50") || !contains(out, "api-work") {
		t.Errorf("log: unexpected output: %q", out)
	}
}

func TestIntegration_Report(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")
	env.logTime("api-work", "2h", "Task A")
	env.logTime("api-work", "1h30m", "Task B")

	out := env.mustRun("report", "--month")
	if !contains(out, "api-work") {
		t.Errorf("report: expected project, got: %q", out)
	}
	// 3.5h × $100 = $350
	if !contains(out, "3.50") {
		t.Errorf("report: expected 3.50h, got: %q", out)
	}
}

func TestIntegration_Export_CSV(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")
	env.logTime("api-work", "1h", "Task")

	out := env.mustRun("export", "--format", "csv")
	if !contains(out, "api-work") {
		t.Errorf("CSV export: expected session, got: %q", out)
	}
	// Must have a header row
	if !contains(out, "project_id") {
		t.Errorf("CSV export: expected header row, got: %q", out)
	}
}

func TestIntegration_Export_JSON(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")
	env.logTime("api-work", "1h30m", "Task")

	out := env.mustRun("export", "--format", "json")
	var sessions []map[string]interface{}
	if err := json.Unmarshal([]byte(out), &sessions); err != nil {
		t.Fatalf("JSON export: invalid JSON: %v\noutput: %s", err, out)
	}
	if len(sessions) != 1 {
		t.Errorf("JSON export: expected 1 session, got %d", len(sessions))
	}
	if sessions[0]["project_id"] != "api-work" {
		t.Errorf("JSON export: unexpected session: %v", sessions[0])
	}
}

func TestIntegration_InvoiceLifecycle(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")
	env.logTime("api-work", "2h", "Sprint 1")
	env.logTime("api-work", "1h", "Sprint 2")

	// Create invoice
	out := env.mustRun("invoice", "create", "--project", "api-work", "--description", "March work")
	if !contains(out, "INV-001") {
		t.Errorf("create: expected invoice number, got: %q", out)
	}

	// List invoices
	out = env.mustRun("invoice", "list")
	if !contains(out, "INV-001") || !contains(out, "Draft") {
		t.Errorf("list: expected draft invoice, got: %q", out)
	}

	// Show invoice
	out = env.mustRun("invoice", "show", "INV-001")
	if !contains(out, "INV-001") || !contains(out, "300") { // 3h × $100
		t.Errorf("show: expected invoice details, got: %q", out)
	}

	// Mark sent
	env.mustRun("invoice", "mark-sent", "INV-001")
	out = env.mustRun("invoice", "list")
	if !contains(out, "Sent") {
		t.Errorf("mark-sent: expected sent status, got: %q", out)
	}

	// Mark paid
	env.mustRun("invoice", "mark-paid", "INV-001")
	out = env.mustRun("invoice", "list")
	if !contains(out, "Paid") {
		t.Errorf("mark-paid: expected paid status, got: %q", out)
	}
}

func TestIntegration_InvoiceDelete_UnmarksSessions(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")
	env.logTime("api-work", "2h", "Sprint 1")

	// Create invoice — sessions get marked billed.
	env.mustRun("invoice", "create", "--project", "api-work", "--description", "March work")

	// Second create fails: no unbilled sessions left.
	_, _, code := env.run("invoice", "create", "--project", "api-work", "--description", "Duplicate")
	if code == 0 {
		t.Error("expected second create to fail (no unbilled sessions)")
	}

	// Delete the first invoice — sessions should be unmarked.
	out := env.mustRun("invoice", "delete", "INV-001", "--force")
	if !contains(out, "INV-001") || !contains(out, "deleted") {
		t.Errorf("delete: unexpected output: %q", out)
	}
	if !contains(out, "unmarked") {
		t.Errorf("delete: expected sessions-unmarked message, got: %q", out)
	}

	// Sessions are now unbilled — a new invoice should pick them up.
	out = env.mustRun("invoice", "create", "--project", "api-work", "--description", "Re-invoice")
	if !contains(out, "INV-") {
		t.Errorf("re-invoice: expected invoice number, got: %q", out)
	}
}

func TestIntegration_InvoiceDelete_SentRequiresForce(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")
	env.logTime("api-work", "1h", "Task")

	env.mustRun("invoice", "create", "--project", "api-work", "--description", "Work")
	env.mustRun("invoice", "mark-sent", "INV-001")

	// Delete without --force should fail
	_, stderr := env.mustFail("invoice", "delete", "INV-001")
	if !contains(stderr, "force") {
		t.Errorf("expected force hint in error, got: %q", stderr)
	}

	// Delete with --force should succeed
	out := env.mustRun("invoice", "delete", "INV-001", "--force")
	if !contains(out, "deleted") {
		t.Errorf("force delete: expected 'deleted', got: %q", out)
	}
}

func TestIntegration_InvoiceDelete_PaidRequiresForce(t *testing.T) {
	env := newTestEnv(t)
	env.addClient("ACME Corp")
	env.addProject("API Work", "acme-corp")
	env.logTime("api-work", "1h", "Task")

	env.mustRun("invoice", "create", "--project", "api-work", "--description", "Work")
	env.mustRun("invoice", "mark-paid", "INV-001")

	_, stderr := env.mustFail("invoice", "delete", "INV-001")
	if !contains(stderr, "force") {
		t.Errorf("expected force hint in error, got: %q", stderr)
	}
}

func TestIntegration_InvoiceDelete_NotFound(t *testing.T) {
	env := newTestEnv(t)
	_, stderr := env.mustFail("invoice", "delete", "INV-999", "--force")
	if !contains(stderr, "not found") && !contains(stderr, "INV-999") {
		t.Errorf("expected not-found error, got: %q", stderr)
	}
}
