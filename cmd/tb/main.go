package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"

	"github.com/devjoaoGustavo/tb/internal/config"
	"github.com/devjoaoGustavo/tb/internal/invoice"
	"github.com/devjoaoGustavo/tb/internal/model"
	"github.com/devjoaoGustavo/tb/internal/numbering"
	"github.com/devjoaoGustavo/tb/internal/store"
	"github.com/spf13/cobra"
)

// version is set at build time via -ldflags "-X main.version=<tag>".
var version = "dev"

func main() {
	if err := newRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:          "tb",
		Short:        "Time Bill — track time and generate invoices",
		Long:         "tb is a CLI tool for freelancers to track time and generate professional invoices.\nAll data is stored locally. No accounts, no cloud.",
		SilenceUsage: true,
		Version:      version,
	}

	root.AddCommand(
		newInitCmd(),
		newConfigCmd(),
		newClientCmd(),
		newProjectCmd(),
		newStartCmd(),
		newStopCmd(),
		newSwitchCmd(),
		newNowCmd(),
		newLogCmd(),
		newCancelCmd(),
		newReportCmd(),
		newInvoiceCmd(),
		newDashboardCmd(),
		newExportCmd(),
	)

	return root
}

// openStore loads config and opens the database.
func openStore() (*store.Store, *config.Config, config.Paths, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, nil, config.Paths{}, err
	}
	paths := config.ResolvePaths(cfg)
	if err := os.MkdirAll(paths.DataDir, 0755); err != nil {
		return nil, nil, paths, fmt.Errorf("creating data dir: %w", err)
	}
	st, err := store.Open(paths.DBFile)
	if err != nil {
		return nil, nil, paths, err
	}
	return st, cfg, paths, nil
}

// --- Helpers ---

// toSlug converts a name like "TechStart Inc." to "techstart-inc".
func toSlug(name string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(name) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			prevDash = false
		} else if !prevDash && b.Len() > 0 {
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.TrimRight(b.String(), "-")
}

// parseAgo parses strings like "2h ago" or plain durations like "30m".
func parseAgo(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimSuffix(s, " ago")
	s = strings.TrimSuffix(s, "ago")
	return time.ParseDuration(strings.TrimSpace(s))
}

// openBrowser opens a file:// URL in the default browser.
func openBrowser(path string) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}
	url := "file://" + absPath
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported OS: %s", runtime.GOOS)
	}
	return cmd.Start()
}

// promptEditor opens $EDITOR with a temp file pre-filled with seed text.
// Lines starting with '#' are stripped from the result.
func promptEditor(seed string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}
	tmp, err := os.CreateTemp("", "tb-*.txt")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.WriteString(seed); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	c := exec.Command(editor, tmpPath)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		return "", err
	}

	data, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", err
	}

	var lines []string
	for _, line := range strings.Split(string(data), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "#") {
			lines = append(lines, line)
		}
	}
	return strings.TrimSpace(strings.Join(lines, "\n")), nil
}

// confirmDelete prints a warning and asks for confirmation unless force is true.
func confirmDelete(label string, force bool) (bool, error) {
	if force {
		return true, nil
	}
	fmt.Printf("Delete %s? [y/N] ", label)
	r := bufio.NewReader(os.Stdin)
	line, _ := r.ReadString('\n')
	return strings.ToLower(strings.TrimSpace(line)) == "y", nil
}

// promptString prints a prompt and reads a line; returns def if input is empty.
func promptString(r *bufio.Reader, label, def string) string {
	if def != "" {
		fmt.Printf("%s [%s]: ", label, def)
	} else {
		fmt.Printf("%s: ", label)
	}
	line, _ := r.ReadString('\n')
	line = strings.TrimSpace(line)
	if line == "" {
		return def
	}
	return line
}

// --- Commands ---

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive first-time setup",
		RunE: func(cmd *cobra.Command, args []string) error {
			paths := config.ResolvePaths(nil)
			if _, err := os.Stat(paths.ConfigFile); err == nil {
				fmt.Printf("Already initialized at %s\n", paths.ConfigFile)
				fmt.Println("Use `tb config edit` to change settings.")
				return nil
			}

			fmt.Println("Welcome to tb! Let's set up your profile.")
			fmt.Println()

			r := bufio.NewReader(os.Stdin)
			cfg := config.DefaultConfig()

			fmt.Println("=== Issuer info ===")
			cfg.Issuer.Name = promptString(r, "Your name", cfg.Issuer.Name)
			cfg.Issuer.Email = promptString(r, "Email", cfg.Issuer.Email)
			cfg.Issuer.Title = promptString(r, "Title/role (optional)", "")
			cfg.Issuer.Phone = promptString(r, "Phone (optional)", "")
			cfg.Issuer.Address = promptString(r, "Address (optional)", "")
			cfg.Issuer.TaxID = promptString(r, "Tax ID / CPF/CNPJ (optional)", "")

			fmt.Println()
			fmt.Println("=== Invoice defaults ===")
			currStr := promptString(r, "Default currency (BRL/USD/EUR)", string(cfg.Invoice.DefaultCurrency))
			cfg.Invoice.DefaultCurrency = model.Currency(strings.ToUpper(currStr))
			cfg.Invoice.Prefix = promptString(r, "Invoice prefix", cfg.Invoice.Prefix)
			dueDaysStr := promptString(r, "Default due days", fmt.Sprintf("%d", cfg.Invoice.DefaultDueDays))
			if n, err := fmt.Sscanf(dueDaysStr, "%d", &cfg.Invoice.DefaultDueDays); n == 0 || err != nil {
				cfg.Invoice.DefaultDueDays = 15
			}
			taxRateStr := promptString(r, "Default tax rate (0-100, e.g. 5 for 5%)", "0")
			var taxPercent float64
			if _, err := fmt.Sscanf(taxRateStr, "%f", &taxPercent); err == nil {
				cfg.Invoice.DefaultTaxRate = taxPercent / 100
			}
			cfg.Invoice.DefaultNotes = promptString(r, "Default invoice notes (optional)", "")

			if err := config.Save(&cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			if err := os.MkdirAll(paths.InvoiceDir, 0755); err != nil {
				return fmt.Errorf("creating invoice dir: %w", err)
			}

			dbPaths := config.ResolvePaths(&cfg)
			if err := os.MkdirAll(dbPaths.DataDir, 0755); err != nil {
				return fmt.Errorf("creating data dir: %w", err)
			}
			st, err := store.Open(dbPaths.DBFile)
			if err != nil {
				return fmt.Errorf("initializing database: %w", err)
			}
			st.Close()

			fmt.Println()
			fmt.Printf("Config saved to %s\n", paths.ConfigFile)
			fmt.Printf("Database created at %s\n", dbPaths.DBFile)
			fmt.Println()
			fmt.Println("Next steps:")
			fmt.Println("  tb client add \"ACME Corp\" --email cfo@acme.com")
			fmt.Println("  tb project add \"API Work\" --client acme-corp --rate 150")
			fmt.Println("  tb start api-work")
			return nil
		},
	}
}

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	show := &cobra.Command{
		Use:   "show",
		Short: "Print current configuration as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(data))
			return nil
		},
	}

	edit := &cobra.Command{
		Use:   "edit",
		Short: "Open config file in $EDITOR",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			paths := config.ResolvePaths(cfg)
			editor := os.Getenv("EDITOR")
			if editor == "" {
				editor = "vi"
			}
			c := exec.Command(editor, paths.ConfigFile)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}

	cmd.AddCommand(show, edit)
	return cmd
}

func newClientCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "Manage clients",
	}

	add := &cobra.Command{
		Use:   "add [name]",
		Short: "Add a new client (interactive when no flags are given)",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			email, _ := cmd.Flags().GetString("email")
			company, _ := cmd.Flags().GetString("company")
			address, _ := cmd.Flags().GetString("address")
			phone, _ := cmd.Flags().GetString("phone")
			taxID, _ := cmd.Flags().GetString("tax-id")

			name := ""
			if len(args) > 0 {
				name = args[0]
			}

			// Interactive mode when name is absent or no flags were set.
			interactive := name == "" || !cmd.Flags().Changed("email") && !cmd.Flags().Changed("company") &&
				!cmd.Flags().Changed("address") && !cmd.Flags().Changed("phone") && !cmd.Flags().Changed("tax-id")

			if interactive {
				r := bufio.NewReader(os.Stdin)
				fmt.Println("New client — press Enter to skip optional fields.")
				fmt.Println()
				name = promptString(r, "Name", name)
				if name == "" {
					return fmt.Errorf("name is required")
				}
				company = promptString(r, "Company", company)
				email = promptString(r, "Billing email", email)
				phone = promptString(r, "Phone", phone)
				address = promptString(r, "Address", address)
				taxID = promptString(r, "Tax ID / CPF/CNPJ", taxID)
			}

			c := model.Client{
				ID:      toSlug(name),
				Name:    name,
				Email:   email,
				Company: company,
				Address: address,
				Phone:   phone,
				TaxID:   taxID,
				Created: time.Now(),
			}
			if err := st.CreateClient(c); err != nil {
				return err
			}
			fmt.Printf("Client %q added (id: %s)\n", c.Name, c.ID)
			return nil
		},
	}
	add.Flags().String("email", "", "billing contact email")
	add.Flags().String("company", "", "company name")
	add.Flags().String("tax-id", "", "CPF/CNPJ")
	add.Flags().String("address", "", "billing address")
	add.Flags().String("phone", "", "phone number")

	list := &cobra.Command{
		Use:   "list",
		Short: "List all clients",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			clients, err := st.ListClients()
			if err != nil {
				return err
			}
			if len(clients) == 0 {
				fmt.Println("No clients yet. Add one with: tb client add <name>")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tCOMPANY\tEMAIL")
			for _, c := range clients {
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", c.ID, c.Name, c.Company, c.Email)
			}
			return w.Flush()
		},
	}

	del := &cobra.Command{
		Use:               "delete <id>",
		Short:             "Delete a client and all their projects, sessions, and invoices",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeClientIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			id := args[0]
			c, err := st.GetClientByID(id)
			if err != nil {
				return err
			}
			force, _ := cmd.Flags().GetBool("force")
			ok, err := confirmDelete(fmt.Sprintf("client %q and all associated data", c.Name), force)
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("Aborted.")
				return nil
			}
			if err := st.DeleteClient(id); err != nil {
				return err
			}
			fmt.Printf("Client %q deleted.\n", c.Name)
			return nil
		},
	}
	del.Flags().BoolP("force", "f", false, "skip confirmation prompt")

	cmd.AddCommand(add, list, del)
	return cmd
}

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	add := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a new project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			clientID, _ := cmd.Flags().GetString("client")
			if clientID == "" {
				return fmt.Errorf("--client flag is required")
			}
			if _, err := st.GetClientByID(clientID); err != nil {
				return err
			}

			rate, _ := cmd.Flags().GetFloat64("rate")
			amount, _ := cmd.Flags().GetFloat64("amount")
			billingTypeStr, _ := cmd.Flags().GetString("type")
			currencyStr, _ := cmd.Flags().GetString("currency")

			name := args[0]
			p := model.Project{
				ID:          toSlug(name),
				Name:        name,
				ClientID:    clientID,
				BillingType: model.BillingType(billingTypeStr),
				HourlyRate:  rate,
				FixedAmount: amount,
				Currency:    model.Currency(strings.ToUpper(currencyStr)),
				Active:      true,
				Created:     time.Now(),
			}
			if err := st.CreateProject(p); err != nil {
				return err
			}
			fmt.Printf("Project %q added (id: %s)\n", p.Name, p.ID)
			return nil
		},
	}
	add.Flags().String("client", "", "client slug (required)")
	add.Flags().Float64("rate", 0, "hourly rate")
	add.Flags().String("type", "hourly", "billing type: hourly or fixed")
	add.Flags().Float64("amount", 0, "fixed amount (when --type fixed)")
	add.Flags().String("currency", "BRL", "currency code")
	add.RegisterFlagCompletionFunc("client", completeClientIDs) //nolint

	list := &cobra.Command{
		Use:   "list",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			clientFilter, _ := cmd.Flags().GetString("client")
			projects, err := st.ListProjects(clientFilter)
			if err != nil {
				return err
			}
			if len(projects) == 0 {
				fmt.Println("No projects yet. Add one with: tb project add <name> --client <id>")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "ID\tNAME\tCLIENT\tTYPE\tRATE\tCURRENCY\tACTIVE")
			for _, p := range projects {
				rate := "-"
				if p.BillingType == model.BillingHourly {
					rate = fmt.Sprintf("%.2f/h", p.HourlyRate)
				} else {
					rate = fmt.Sprintf("%.2f (fixed)", p.FixedAmount)
				}
				active := "yes"
				if !p.Active {
					active = "no"
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
					p.ID, p.Name, p.ClientID, string(p.BillingType), rate, string(p.Currency), active)
			}
			return w.Flush()
		},
	}
	list.Flags().String("client", "", "filter by client slug")
	list.RegisterFlagCompletionFunc("client", completeClientIDs) //nolint

	del := &cobra.Command{
		Use:               "delete <id>",
		Short:             "Delete a project and all its sessions",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProjectIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			id := args[0]
			p, err := st.GetProjectByID(id)
			if err != nil {
				return err
			}
			force, _ := cmd.Flags().GetBool("force")
			ok, err := confirmDelete(fmt.Sprintf("project %q and all its sessions", p.Name), force)
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("Aborted.")
				return nil
			}
			if err := st.DeleteProject(id); err != nil {
				return err
			}
			fmt.Printf("Project %q deleted.\n", p.Name)
			return nil
		},
	}
	del.Flags().BoolP("force", "f", false, "skip confirmation prompt")

	cmd.AddCommand(add, list, del)
	return cmd
}

func newStartCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "start <project>",
		Short:             "Start a timer for a project",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProjectIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			active, err := st.ActiveSession()
			if err != nil {
				return err
			}
			if active != nil {
				return fmt.Errorf("timer already running for project %q — stop it first with `tb stop`", active.ProjectID)
			}

			projectID := args[0]
			if _, err := st.GetProjectByID(projectID); err != nil {
				return err
			}

			startTime := time.Now()
			if atFlag, _ := cmd.Flags().GetString("at"); atFlag != "" {
				ago, err := parseAgo(atFlag)
				if err != nil {
					return fmt.Errorf("invalid --at value %q: %w", atFlag, err)
				}
				startTime = time.Now().Add(-ago)
			}

			sess := model.Session{
				ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
				ProjectID: projectID,
				Start:     startTime,
			}
			if err := st.CreateSession(sess); err != nil {
				return err
			}
			fmt.Printf("Timer started for %q at %s\n", projectID, startTime.Format("15:04"))
			return nil
		},
	}
	cmd.Flags().String("at", "", `start time, e.g. "2h ago"`)
	return cmd
}

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "Stop the current timer",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			note, _ := cmd.Flags().GetString("note")
			return stopActive(st, note)
		},
	}
	cmd.Flags().String("note", "", "session note")
	return cmd
}

func stopActive(st *store.Store, note string) error {
	active, err := st.ActiveSession()
	if err != nil {
		return err
	}
	if active == nil {
		return fmt.Errorf("no timer running")
	}
	active.End = time.Now()
	active.Note = note
	if err := st.UpdateSession(*active); err != nil {
		return err
	}
	dur := active.Duration()
	fmt.Printf("Stopped %q — %.2f hours (%.0f min)\n", active.ProjectID, dur.Hours(), dur.Minutes())
	return nil
}

func newSwitchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "switch <project>",
		Short:             "Stop current timer and start a new one",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeProjectIDs,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			if err := stopActive(st, ""); err != nil && err.Error() != "no timer running" {
				return err
			}

			projectID := args[0]
			if _, err := st.GetProjectByID(projectID); err != nil {
				return err
			}

			sess := model.Session{
				ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
				ProjectID: projectID,
				Start:     time.Now(),
			}
			if err := st.CreateSession(sess); err != nil {
				return err
			}
			fmt.Printf("Switched to %q\n", projectID)
			return nil
		},
	}
	return cmd
}

func newNowCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "now",
		Aliases: []string{"status"},
		Short:   "Show active project and today's time",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			active, err := st.ActiveSession()
			if err != nil {
				return err
			}
			if active == nil {
				fmt.Println("No timer running.")
				return nil
			}

			dur := active.Duration()
			fmt.Printf("Running: %s  (started %s, elapsed %.0fm)\n",
				active.ProjectID,
				active.Start.Format("15:04"),
				dur.Minutes(),
			)

			now := time.Now()
			startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
			sessions, err := st.ListSessions(active.ProjectID, startOfDay, now)
			if err != nil {
				return err
			}
			var totalHours float64
			for _, s := range sessions {
				totalHours += s.Hours()
			}
			fmt.Printf("Today: %.2fh on %s\n", totalHours, active.ProjectID)
			return nil
		},
	}
}

func newLogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "log <project> <duration> [note]",
		Short: "Manually log time for a project",
		Args:  cobra.RangeArgs(2, 3),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			if len(args) == 0 {
				return completeProjectIDs(cmd, args, toComplete)
			}
			return nil, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			projectID := args[0]
			if _, err := st.GetProjectByID(projectID); err != nil {
				return err
			}

			dur, err := time.ParseDuration(args[1])
			if err != nil {
				return fmt.Errorf("invalid duration %q: %w", args[1], err)
			}

			note := ""
			if len(args) == 3 {
				note = args[2]
			}

			end := time.Now()
			sess := model.Session{
				ID:        fmt.Sprintf("%d", time.Now().UnixNano()),
				ProjectID: projectID,
				Start:     end.Add(-dur),
				End:       end,
				Note:      note,
			}
			if err := st.CreateSession(sess); err != nil {
				return err
			}
			fmt.Printf("Logged %.2fh for %q\n", dur.Hours(), projectID)
			return nil
		},
	}
}

func newCancelCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "cancel",
		Short: "Discard the current running session",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			active, err := st.ActiveSession()
			if err != nil {
				return err
			}
			if active == nil {
				return fmt.Errorf("no timer running")
			}
			if err := st.DeleteSession(active.ID); err != nil {
				return err
			}
			fmt.Printf("Cancelled session for %q\n", active.ProjectID)
			return nil
		},
	}
}

func newReportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Show a time report",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			from, to, err := resolveDateRange(cmd)
			if err != nil {
				return err
			}

			clientFilter, _ := cmd.Flags().GetString("client")
			projectFilter, _ := cmd.Flags().GetString("project")

			var sessions []model.Session
			if clientFilter != "" {
				sessions, err = st.ListSessionsByClient(clientFilter, from, to)
			} else {
				sessions, err = st.ListSessions(projectFilter, from, to)
			}
			if err != nil {
				return err
			}

			if len(sessions) == 0 {
				fmt.Println("No sessions found for the given filters.")
				return nil
			}

			// Group by project
			type projectEntry struct {
				projectID string
				hours     float64
			}
			byProject := map[string]float64{}
			for _, s := range sessions {
				if s.End.IsZero() {
					continue
				}
				byProject[s.ProjectID] += s.Hours()
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "PROJECT\tCLIENT\tHOURS\tEARNINGS\tPERIOD")
			for projectID, hours := range byProject {
				proj, err := st.GetProjectByID(projectID)
				if err != nil {
					continue
				}
				var earnings float64
				if proj.BillingType == model.BillingHourly {
					earnings = hours * proj.HourlyRate
				} else {
					earnings = proj.FixedAmount
				}
				period := ""
				if !from.IsZero() && !to.IsZero() {
					period = fmt.Sprintf("%s — %s", from.Format("2006-01-02"), to.Format("2006-01-02"))
				}
				fmt.Fprintf(w, "%s\t%s\t%.2f\t%.2f %s\t%s\n",
					proj.ID, proj.ClientID, hours, earnings, string(proj.Currency), period)
			}
			return w.Flush()
		},
	}
	cmd.Flags().Bool("week", false, "show this week")
	cmd.Flags().Bool("month", false, "show this month")
	cmd.Flags().String("project", "", "filter by project slug")
	cmd.Flags().String("client", "", "filter by client slug")
	cmd.Flags().String("from", "", "start date (YYYY-MM-DD)")
	cmd.Flags().String("to", "", "end date (YYYY-MM-DD)")
	cmd.RegisterFlagCompletionFunc("client", completeClientIDs)   //nolint
	cmd.RegisterFlagCompletionFunc("project", completeProjectIDs) //nolint
	return cmd
}

func resolveDateRange(cmd *cobra.Command) (time.Time, time.Time, error) {
	now := time.Now()

	week, _ := cmd.Flags().GetBool("week")
	month, _ := cmd.Flags().GetBool("month")
	fromStr, _ := cmd.Flags().GetString("from")
	toStr, _ := cmd.Flags().GetString("to")

	if week {
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		from := now.AddDate(0, 0, -(weekday - 1))
		from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
		to := from.AddDate(0, 0, 6)
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())
		return from, to, nil
	}

	if month {
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to := from.AddDate(0, 1, -1)
		to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, to.Location())
		return from, to, nil
	}

	var from, to time.Time
	if fromStr != "" {
		t, err := time.Parse("2006-01-02", fromStr)
		if err != nil {
			return from, to, fmt.Errorf("invalid --from date: %w", err)
		}
		from = t
	}
	if toStr != "" {
		t, err := time.Parse("2006-01-02", toStr)
		if err != nil {
			return from, to, fmt.Errorf("invalid --to date: %w", err)
		}
		to = time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 0, t.Location())
	}
	return from, to, nil
}

func newInvoiceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "invoice",
		Short: "Manage invoices",
	}

	create := &cobra.Command{
		Use:   "create",
		Short: "Create an invoice from tracked sessions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvoiceCreate(cmd, false)
		},
	}
	create.Flags().String("project", "", "project slug (required)")
	create.Flags().String("period", "", "billing period, e.g. 2026-02")
	create.Flags().Bool("fixed", false, "use fixed billing amount")
	create.Flags().StringP("description", "d", "", "invoice description / notes (opens $EDITOR if omitted)")
	create.Flags().String("due", "", "due date override (YYYY-MM-DD)")
	create.RegisterFlagCompletionFunc("project", completeProjectIDs) //nolint

	preview := &cobra.Command{
		Use:   "preview",
		Short: "Preview an invoice without saving",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInvoiceCreate(cmd, true)
		},
	}
	preview.Flags().String("project", "", "project slug (required)")
	preview.Flags().String("period", "", "billing period, e.g. 2026-02")
	preview.Flags().StringP("description", "d", "", "invoice description / notes (opens $EDITOR if omitted)")
	preview.Flags().String("due", "", "due date override (YYYY-MM-DD)")
	preview.RegisterFlagCompletionFunc("project", completeProjectIDs) //nolint

	list := &cobra.Command{
		Use:   "list",
		Short: "List invoices",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			statusFilter, _ := cmd.Flags().GetString("status")
			invoices, err := st.ListInvoices(statusFilter)
			if err != nil {
				return err
			}
			if len(invoices) == 0 {
				fmt.Println("No invoices found.")
				return nil
			}

			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			fmt.Fprintln(w, "NUMBER\tCLIENT\tSTATUS\tTOTAL\tISSUED\tDUE")
			for _, inv := range invoices {
				issued := "-"
				if !inv.IssuedAt.IsZero() {
					issued = inv.IssuedAt.Format("2006-01-02")
				}
				due := "-"
				if !inv.DueAt.IsZero() {
					due = inv.DueAt.Format("2006-01-02")
				}
				fmt.Fprintf(w, "%s\t%s\t%s\t%.2f %s\t%s\t%s\n",
					inv.Number, inv.ClientID, string(inv.Status),
					inv.Total, string(inv.Currency), issued, due)
			}
			return w.Flush()
		},
	}
	list.Flags().String("status", "", "filter by status: draft, sent, paid")

	show := &cobra.Command{
		Use:               "show <number>",
		Short:             "Show invoice details",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeInvoiceNumbers,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			inv, err := st.GetInvoiceByNumber(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("Invoice:  %s\n", inv.Number)
			fmt.Printf("Status:   %s\n", inv.Status)
			fmt.Printf("Client:   %s\n", inv.ClientID)
			fmt.Printf("Project:  %s\n", inv.ProjectID)
			if !inv.IssuedAt.IsZero() {
				fmt.Printf("Issued:   %s\n", inv.IssuedAt.Format("2006-01-02"))
			}
			if !inv.DueAt.IsZero() {
				fmt.Printf("Due:      %s\n", inv.DueAt.Format("2006-01-02"))
			}
			if !inv.PaidAt.IsZero() {
				fmt.Printf("Paid:     %s\n", inv.PaidAt.Format("2006-01-02"))
			}
			if !inv.PeriodStart.IsZero() {
				fmt.Printf("Period:   %s — %s\n",
					inv.PeriodStart.Format("2006-01-02"),
					inv.PeriodEnd.Format("2006-01-02"))
			}
			fmt.Println()

			sym := invoice.CurrencySymbol(inv.Currency)
			w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
			if inv.BillingType == model.BillingHourly {
				fmt.Fprintln(w, "DATE\tDESCRIPTION\tHOURS\tRATE\tAMOUNT")
				for _, li := range inv.LineItems {
					fmt.Fprintf(w, "%s\t%s\t%.2f\t%s %.2f\t%s %.2f\n",
						li.Date, li.Description, li.Hours, sym, li.Rate, sym, li.Amount)
				}
			} else {
				fmt.Fprintln(w, "DESCRIPTION\tAMOUNT")
				for _, li := range inv.LineItems {
					fmt.Fprintf(w, "%s\t%s %.2f\n", li.Description, sym, li.Amount)
				}
			}
			w.Flush()

			fmt.Println()
			fmt.Printf("Subtotal: %s %.2f\n", sym, inv.Subtotal)
			if inv.TaxRate > 0 {
				fmt.Printf("Tax %.0f%%: %s %.2f\n", inv.TaxRate*100, sym, inv.Tax)
			}
			fmt.Printf("Total:    %s %.2f\n", sym, inv.Total)
			if inv.Notes != "" {
				fmt.Printf("\nNotes: %s\n", inv.Notes)
			}
			return nil
		},
	}

	markSent := &cobra.Command{
		Use:               "mark-sent <number>",
		Short:             "Mark an invoice as sent",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeInvoiceNumbers,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			if err := st.UpdateInvoiceStatus(args[0], model.InvoiceSent, time.Time{}); err != nil {
				return err
			}
			fmt.Printf("Invoice %s marked as sent\n", args[0])
			return nil
		},
	}

	markPaid := &cobra.Command{
		Use:               "mark-paid <number>",
		Short:             "Mark an invoice as paid",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeInvoiceNumbers,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			paidAt := time.Now()
			if dateStr, _ := cmd.Flags().GetString("date"); dateStr != "" {
				t, err := time.Parse("2006-01-02", dateStr)
				if err != nil {
					return fmt.Errorf("invalid --date: %w", err)
				}
				paidAt = t
			}

			if err := st.UpdateInvoiceStatus(args[0], model.InvoicePaid, paidAt); err != nil {
				return err
			}
			fmt.Printf("Invoice %s marked as paid (%s)\n", args[0], paidAt.Format("2006-01-02"))
			return nil
		},
	}
	markPaid.Flags().String("date", "", "payment date (YYYY-MM-DD)")

	open := &cobra.Command{
		Use:               "open <number>",
		Short:             "Open invoice HTML in the browser",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeInvoiceNumbers,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, cfg, paths, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			inv, err := st.GetInvoiceByNumber(args[0])
			if err != nil {
				return err
			}

			htmlPath := filepath.Join(paths.InvoiceDir, inv.Number+".html")
			if _, err := os.Stat(htmlPath); os.IsNotExist(err) {
				fmt.Printf("HTML file not found at %s — regenerating...\n", htmlPath)
				proj, err := st.GetProjectByID(inv.ProjectID)
				if err != nil {
					return err
				}
				client, err := st.GetClientByID(inv.ClientID)
				if err != nil {
					return err
				}
				if err := invoice.RenderToFile(htmlPath, inv, proj, client, cfg.Issuer); err != nil {
					return err
				}
			}

			return openBrowser(htmlPath)
		},
	}

	del := &cobra.Command{
		Use:               "delete <number>",
		Short:             "Delete an invoice",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completeInvoiceNumbers,
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, paths, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			number := args[0]
			inv, err := st.GetInvoiceByNumber(number)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")

			// Require --force for sent/paid invoices to prevent accidental loss.
			if (inv.Status == model.InvoiceSent || inv.Status == model.InvoicePaid) && !force {
				return fmt.Errorf("invoice %s has status %q — use --force to delete it", number, inv.Status)
			}

			ok, err := confirmDelete(fmt.Sprintf("invoice %s (status: %s)", number, inv.Status), force)
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("Aborted.")
				return nil
			}

			// Collect session IDs from line items so we can unmark them.
			var sessionIDs []string
			for _, li := range inv.LineItems {
				if li.SessionID != "" {
					sessionIDs = append(sessionIDs, li.SessionID)
				}
			}

			if err := st.DeleteInvoice(number); err != nil {
				return err
			}

			// Unmark sessions so they can be re-invoiced.
			if err := st.UnmarkSessionsBilled(sessionIDs); err != nil {
				return err
			}

			htmlPath := filepath.Join(paths.InvoiceDir, number+".html")
			if rmErr := os.Remove(htmlPath); rmErr == nil {
				fmt.Printf("Invoice %s deleted (HTML file removed, %d session(s) unmarked).\n", number, len(sessionIDs))
			} else {
				fmt.Printf("Invoice %s deleted (%d session(s) unmarked).\n", number, len(sessionIDs))
			}
			return nil
		},
	}
	del.Flags().BoolP("force", "f", false, "skip confirmation prompt")

	cmd.AddCommand(create, preview, list, show, markSent, markPaid, open, del)
	return cmd
}

func parsePeriod(period string) (time.Time, time.Time, error) {
	now := time.Now()
	if period == "" {
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		end := start.AddDate(0, 1, -1)
		end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, end.Location())
		return start, end, nil
	}
	t, err := time.Parse("2006-01", period)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --period %q, expected YYYY-MM: %w", period, err)
	}
	start := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, now.Location())
	end := start.AddDate(0, 1, -1)
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, end.Location())
	return start, end, nil
}

func runInvoiceCreate(cmd *cobra.Command, preview bool) error {
	st, cfg, paths, err := openStore()
	if err != nil {
		return err
	}
	defer st.Close()

	projectID, _ := cmd.Flags().GetString("project")
	if projectID == "" {
		return fmt.Errorf("--project flag is required")
	}
	useFixed, _ := cmd.Flags().GetBool("fixed")
	periodStr, _ := cmd.Flags().GetString("period")
	description, _ := cmd.Flags().GetString("description")
	dueStr, _ := cmd.Flags().GetString("due")

	if description == "" {
		var err error
		description, err = promptEditor("# Invoice description — add notes for the client (lines starting with # are ignored)\n")
		if err != nil {
			return fmt.Errorf("opening editor: %w", err)
		}
	}

	proj, err := st.GetProjectByID(projectID)
	if err != nil {
		return err
	}
	client, err := st.GetClientByID(proj.ClientID)
	if err != nil {
		return err
	}

	periodStart, periodEnd, err := parsePeriod(periodStr)
	if err != nil {
		return err
	}

	sessions, err := st.ListSessions(projectID, periodStart, periodEnd)
	if err != nil {
		return err
	}

	var lineItems []model.LineItem
	var sessionIDs []string

	billingType := proj.BillingType
	if useFixed {
		billingType = model.BillingFixed
	}

	if billingType == model.BillingHourly {
		for _, s := range sessions {
			if s.Billed || s.End.IsZero() {
				continue
			}
			li := model.LineItem{
				Description: s.Note,
				Date:        s.Start.Format("2006-01-02"),
				Hours:       s.Hours(),
				Rate:        proj.HourlyRate,
				Amount:      s.Hours() * proj.HourlyRate,
				SessionID:   s.ID,
			}
			if li.Description == "" {
				li.Description = proj.Name
			}
			lineItems = append(lineItems, li)
			sessionIDs = append(sessionIDs, s.ID)
		}
	} else {
		desc := description
		if desc == "" {
			desc = proj.Name
		}
		lineItems = []model.LineItem{
			{
				Description: desc,
				Amount:      proj.FixedAmount,
			},
		}
	}

	if len(lineItems) == 0 {
		return fmt.Errorf("no unbilled sessions found for %q in period %s — %s",
			projectID, periodStart.Format("2006-01-02"), periodEnd.Format("2006-01-02"))
	}

	var subtotal float64
	for _, li := range lineItems {
		subtotal += li.Amount
	}
	taxRate := cfg.Invoice.DefaultTaxRate
	tax := subtotal * taxRate
	total := subtotal + tax

	var invNumber string
	var seqNum int
	if preview {
		seqNum, err = st.PeekClientSequence(client.ID)
	} else {
		seqNum, err = st.NextClientSequence(client.ID)
	}
	if err != nil {
		return err
	}
	invNumber = numbering.FormatNumber(cfg, seqNum, client.ID)

	now := time.Now()
	dueAt := now.AddDate(0, 0, cfg.Invoice.DefaultDueDays)
	if dueStr != "" {
		t, err := time.Parse("2006-01-02", dueStr)
		if err != nil {
			return fmt.Errorf("invalid --due date %q: %w", dueStr, err)
		}
		dueAt = t
	}

	notes := cfg.Invoice.DefaultNotes
	if description != "" {
		notes = description
	}

	inv := model.Invoice{
		ID:          fmt.Sprintf("%d", now.UnixNano()),
		Number:      invNumber,
		ClientID:    client.ID,
		ProjectID:   proj.ID,
		Status:      model.InvoiceDraft,
		BillingType: billingType,
		Currency:    proj.Currency,
		LineItems:   lineItems,
		Subtotal:    subtotal,
		Tax:         tax,
		TaxRate:     taxRate,
		Total:       total,
		IssuedAt:    now,
		DueAt:       dueAt,
		Notes:       notes,
		PeriodStart: periodStart,
		PeriodEnd:   periodEnd,
		Created:     now,
	}

	if preview {
		tmpFile, err := os.CreateTemp("", "tb-invoice-*.html")
		if err != nil {
			return err
		}
		tmpPath := tmpFile.Name()
		tmpFile.Close()

		if err := invoice.RenderToFile(tmpPath, inv, proj, client, cfg.Issuer); err != nil {
			return err
		}
		fmt.Printf("Preview: %s\n", tmpPath)
		return openBrowser(tmpPath)
	}

	htmlPath := filepath.Join(paths.InvoiceDir, inv.Number+".html")
	if err := invoice.RenderToFile(htmlPath, inv, proj, client, cfg.Issuer); err != nil {
		return err
	}
	if err := st.CreateInvoice(inv); err != nil {
		return err
	}
	if err := st.MarkSessionsBilled(sessionIDs); err != nil {
		return err
	}

	fmt.Printf("Invoice %s created (draft) — saved to %s\n", inv.Number, htmlPath)
	return nil
}

func newDashboardCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "dashboard",
		Short: "Interactive TUI dashboard",
		RunE: func(cmd *cobra.Command, args []string) error {
			return fmt.Errorf("dashboard is not yet implemented")
		},
	}
}

func newExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export data",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, _, _, err := openStore()
			if err != nil {
				return err
			}
			defer st.Close()

			format, _ := cmd.Flags().GetString("format")
			monthStr, _ := cmd.Flags().GetString("month")

			var from, to time.Time
			if monthStr != "" {
				t, err := time.Parse("2006-01", monthStr)
				if err != nil {
					return fmt.Errorf("invalid --month %q, expected YYYY-MM: %w", monthStr, err)
				}
				from = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.Local)
				to = from.AddDate(0, 1, -1)
				to = time.Date(to.Year(), to.Month(), to.Day(), 23, 59, 59, 0, time.Local)
			}

			sessions, err := st.ListSessions("", from, to)
			if err != nil {
				return err
			}

			switch format {
			case "json":
				enc := json.NewEncoder(os.Stdout)
				enc.SetIndent("", "  ")
				return enc.Encode(sessions)
			default: // csv
				w := csv.NewWriter(os.Stdout)
				w.Write([]string{"id", "project_id", "start", "end", "hours", "note", "billed"}) //nolint
				for _, s := range sessions {
					endStr := ""
					if !s.End.IsZero() {
						endStr = s.End.Format(time.RFC3339)
					}
					billed := "false"
					if s.Billed {
						billed = "true"
					}
					w.Write([]string{ //nolint
						s.ID, s.ProjectID,
						s.Start.Format(time.RFC3339), endStr,
						fmt.Sprintf("%.4f", s.Hours()),
						s.Note, billed,
					})
				}
				w.Flush()
				return w.Error()
			}
		},
	}
	cmd.Flags().String("format", "csv", "output format: csv or json")
	cmd.Flags().String("month", "", "filter by month, e.g. 2026-02")
	return cmd
}

// --- Shell completion helpers ---

func completeProjectIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	st, _, _, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer st.Close()
	projects, err := st.ListProjects("")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var ids []string
	for _, p := range projects {
		ids = append(ids, p.ID+"\t"+p.Name)
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}

func completeClientIDs(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	st, _, _, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer st.Close()
	clients, err := st.ListClients()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var ids []string
	for _, c := range clients {
		ids = append(ids, c.ID+"\t"+c.Name)
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}

func completeInvoiceNumbers(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	st, _, _, err := openStore()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	defer st.Close()
	invoices, err := st.ListInvoices("")
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	var numbers []string
	for _, inv := range invoices {
		numbers = append(numbers, inv.Number+"\t"+string(inv.Status)+" • "+inv.ClientID)
	}
	return numbers, cobra.ShellCompDirectiveNoFileComp
}
