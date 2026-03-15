package cli

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"os"
	"strings"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ── Styles ────────────────────────────────────────────────────────────────────

var (
	titleStyle     = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	questionStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("255"))
	hintStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("243")).Italic(true)
	selectedStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("99"))
	dimChoiceStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	stepStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("243"))
	okStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	errorStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
)

// ── Steps ─────────────────────────────────────────────────────────────────────

type stepID int

const (
	stepSuperAdminEmail stepID = iota
	stepVAPIDEmail
	stepNodeURL
	stepIsPrimary
	stepPrimaryNodeURL // only when IS_PRIMARY=false
	stepDBPath
	stepAPIURL
	stepDone
)

type stepMeta struct {
	question    string
	hint        string
	placeholder string
	def         string
	isBool      bool
}

var steps = map[stepID]stepMeta{
	stepSuperAdminEmail: {
		question:    "Super Admin Email",
		hint:        "Email address for the superadmin account (created on first start)",
		placeholder: "superadmin@seashell.local",
		def:         "superadmin@seashell.local",
	},
	stepVAPIDEmail: {
		question:    "VAPID Contact Email",
		hint:        "Contact email for Web Push (leave blank to skip push notifications)",
		placeholder: "mailto:admin@seashell.local",
		def:         "mailto:admin@seashell.local",
	},
	stepNodeURL: {
		question:    "Node URL",
		hint:        "This node's publicly reachable base URL (no trailing slash)",
		placeholder: "https://primary.example.com",
	},
	stepIsPrimary: {
		question: "Is this the primary (genesis) node?",
		hint:     "Use ← → or Y/N to choose, then Enter to confirm",
		isBool:   true,
	},
	stepPrimaryNodeURL: {
		question:    "Primary Node URL",
		hint:        "URL of the primary node this node should connect to",
		placeholder: "https://primary.example.com",
	},
	stepDBPath: {
		question:    "Database Path",
		hint:        "Directory where blockchain and API data are stored",
		placeholder: "./db",
		def:         "./db",
	},
	stepAPIURL: {
		question:    "API URL",
		hint:        "Full URL of this API, used by the Next.js UI",
		placeholder: "http://localhost:8080",
		def:         "http://localhost:8080",
	},
}

// activeSteps returns the ordered question sequence based on the isPrimary toggle.
func activeSteps(isPrimary bool) []stepID {
	seq := []stepID{
		stepSuperAdminEmail,
		stepVAPIDEmail,
		stepNodeURL,
		stepIsPrimary,
	}
	if !isPrimary {
		seq = append(seq, stepPrimaryNodeURL)
	}
	return append(seq, stepDBPath, stepAPIURL)
}

// ── Model ─────────────────────────────────────────────────────────────────────

type genEnvModel struct {
	step      stepID
	input     textinput.Model
	isPrimary bool
	outPath   string
	answers   map[stepID]string
	err       string
	saved     bool
}

func newGenEnvModel(outPath string) genEnvModel {
	ti := textinput.New()
	meta := steps[stepSuperAdminEmail]
	ti.Placeholder = meta.placeholder
	ti.Focus()
	return genEnvModel{
		step:      stepSuperAdminEmail,
		input:     ti,
		isPrimary: true,
		outPath:   outPath,
		answers:   make(map[stepID]string),
	}
}

func (m genEnvModel) Init() tea.Cmd { return textinput.Blink }

// nextStep looks up the current step in the active sequence and returns the one after it.
func (m genEnvModel) nextStep() stepID {
	seq := activeSteps(m.isPrimary)
	for i, id := range seq {
		if id == m.step && i+1 < len(seq) {
			return seq[i+1]
		}
	}
	return stepDone
}

func (m *genEnvModel) commitStep() {
	if steps[m.step].isBool {
		if m.isPrimary {
			m.answers[m.step] = "true"
		} else {
			m.answers[m.step] = "false"
		}
		return
	}
	val := strings.TrimSpace(m.input.Value())
	if val == "" {
		val = steps[m.step].def
	}
	m.answers[m.step] = val
}

func (m *genEnvModel) loadStep(s stepID) {
	meta := steps[s]
	ti := textinput.New()
	ti.Placeholder = meta.placeholder
	ti.Focus()
	m.input = ti
}

func (m genEnvModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.step == stepDone {
		return m, tea.Quit
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit

		case tea.KeyEnter:
			m.commitStep()
			next := m.nextStep()
			if next == stepDone {
				m.step = stepDone
				if err := m.writeEnv(); err != nil {
					m.err = err.Error()
				} else {
					m.saved = true
				}
				return m, tea.Quit
			}
			m.step = next
			m.loadStep(next)
			return m, textinput.Blink

		case tea.KeyLeft, tea.KeyRight:
			if steps[m.step].isBool {
				m.isPrimary = !m.isPrimary
				return m, nil
			}

		default:
			if steps[m.step].isBool {
				switch strings.ToLower(msg.String()) {
				case "y":
					m.isPrimary = true
				case "n":
					m.isPrimary = false
				}
				return m, nil
			}
		}
	}

	if !steps[m.step].isBool {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m genEnvModel) View() string {
	if m.step == stepDone {
		if m.saved {
			return "\n" + okStyle.Render("✓ .env written to "+m.outPath) + "\n\n"
		}
		return "\n" + errorStyle.Render("✗ Error: "+m.err) + "\n\n"
	}

	meta := steps[m.step]
	seq := activeSteps(m.isPrimary)
	num := 1
	for i, id := range seq {
		if id == m.step {
			num = i + 1
			break
		}
	}

	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n", titleStyle.Render("SeaShell — Generate .env"))
	fmt.Fprintf(&b, "%s\n\n", stepStyle.Render(fmt.Sprintf("Step %d of %d", num, len(seq))))
	fmt.Fprintf(&b, "%s\n", questionStyle.Render(meta.question))
	fmt.Fprintf(&b, "%s\n\n", hintStyle.Render(meta.hint))

	if meta.isBool {
		yes := dimChoiceStyle.Render("  Yes")
		no := dimChoiceStyle.Render("  No")
		if m.isPrimary {
			yes = selectedStyle.Render("▶ Yes")
		} else {
			no = selectedStyle.Render("▶ No")
		}
		fmt.Fprintf(&b, "%s    %s\n\n", yes, no)
	} else {
		fmt.Fprintf(&b, "%s\n\n", m.input.View())
	}

	fmt.Fprintf(&b, "%s\n", hintStyle.Render("Enter to confirm · Esc to cancel"))
	return b.String()
}

// ── .env generation ───────────────────────────────────────────────────────────

func (m *genEnvModel) writeEnv() error {
	jwtSecret, err := randomHex(32)
	if err != nil {
		return err
	}
	nodeSecret, err := randomHex(32)
	if err != nil {
		return err
	}
	dbEncKey, err := randomHex(32)
	if err != nil {
		return err
	}
	superAdminPassword, err := randomPassword(32)
	if err != nil {
		return err
	}

	vapidPrivate, vapidPublic := "", ""
	if m.answers[stepVAPIDEmail] != "" {
		vapidPrivate, vapidPublic, err = webpush.GenerateVAPIDKeys()
		if err != nil {
			return fmt.Errorf("generate VAPID keys: %w", err)
		}
	}

	isPrimary := m.answers[stepIsPrimary] != "false"

	lines := []string{
		"# ── Secrets ─────────────────────────────────────────────────────",
		"JWT_SECRET='" + jwtSecret + "'",
		"NODE_SECRET='" + nodeSecret + "'",
		"DB_ENCRYPTION_KEY='" + dbEncKey + "'",
		"",
		"# ── Super Admin ──────────────────────────────────────────────────",
		"SUPERADMIN_EMAIL='" + m.answers[stepSuperAdminEmail] + "'",
		"SUPERADMIN_PASSWORD='" + superAdminPassword + "'",
		"",
		"# ── Node ─────────────────────────────────────────────────────────",
		"NODE_URL='" + m.answers[stepNodeURL] + "'",
		fmt.Sprintf("IS_PRIMARY=%v", isPrimary),
	}

	if !isPrimary {
		lines = append(lines, "PRIMARY_NODE_URL='"+m.answers[stepPrimaryNodeURL]+"'")
	}

	lines = append(lines,
		"",
		"# ── Database ─────────────────────────────────────────────────────",
		"DB_PATH='"+m.answers[stepDBPath]+"'",
		"",
		"# ── Web Push (VAPID) ─────────────────────────────────────────────",
		"VAPID_EMAIL='"+m.answers[stepVAPIDEmail]+"'",
		"VAPID_PUBLIC_KEY='"+vapidPublic+"'",
		"VAPID_PRIVATE_KEY='"+vapidPrivate+"'",
		"",
		"# ── API ──────────────────────────────────────────────────────────",
		"API_URL='"+m.answers[stepAPIURL]+"'",
		"",
		"# ── Others ───────────────────────────────────────────────────────",
		"PORT=8080",
		"UI_PORT=3000",
		"ACCESS_TOKEN_MINUTES=15",
		"REFRESH_TOKEN_DAYS=7",
	)

	return os.WriteFile(m.outPath, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

const passwordChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789@!-_+=~^"

// randomPassword generates a cryptographically random password of the given
// length using alphanumeric characters plus the symbols @!-_+=~^.
func randomPassword(length int) (string, error) {
	charset := big.NewInt(int64(len(passwordChars)))
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, charset)
		if err != nil {
			return "", err
		}
		b[i] = passwordChars[n.Int64()]
	}
	return string(b), nil
}

// randomHex generates n random bytes and returns them hex-encoded.
func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ── Entry point ───────────────────────────────────────────────────────────────

// RunGenEnv is the entry point for the `genenv` subcommand.
func RunGenEnv(outPath string) {
	if outPath == "" {
		outPath = ".env"
	}
	if _, err := os.Stat(outPath); err == nil {
		fmt.Printf("Warning: %s already exists and will be overwritten.\nPress Enter to continue or Ctrl-C to cancel...\n", outPath)
		fmt.Scanln()
	}
	m := newGenEnvModel(outPath)
	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
