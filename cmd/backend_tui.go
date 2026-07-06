package cmd

// NOTE: TUI requires a modern terminal with ANSI support. On Windows use
// Windows Terminal / VS Code integrated terminal / PowerShell 7. Legacy
// conhost may garble output; CI and older terminals must use
// --backend=file --password-file.

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// ErrCancelled is returned when the user aborts backend selection with
// q, Esc, or Ctrl+C.
var ErrCancelled = errors.New("backend selection cancelled")

// ErrNonInteractive is returned when the caller invokes the selector on a
// non-TTY (piped stdin, CI, etc.). Callers should route the user to the
// non-interactive flags surfaced in the stderr hint.
var ErrNonInteractive = errors.New("cannot show backend selector: not a TTY")

// backendChoice is the outcome of a successful selection. Verbose reflects
// the union of the caller-supplied flag and any inline `?` toggle the user
// performed inside the TUI.
type backendChoice struct {
	Kind    BackendKind
	Verbose bool
}

// runBackendSelectorFn is the test seam mirroring cmd/root.go::promptPasswordFn.
// Production wiring points to runBackendSelector; tests reassign it to a
// deterministic stub before invoking the command under test.
var runBackendSelectorFn = runBackendSelector

// backendItem adapts a BackendStatus into bubbles/list.Item. The list
// delegate renders it via Title/Description; FilterValue is unused because
// the list runs with filtering disabled but the interface still requires it.
type backendItem struct {
	status BackendStatus
}

var (
	availableMark   = lipgloss.NewStyle().Foreground(lipgloss.Color("2")).Render("[✓]")
	unavailableMark = lipgloss.NewStyle().Foreground(lipgloss.Color("1")).Render("[✗]")
	reasonStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

// Title renders the "[✓|✗] Name" prefix. The mark is ANSI-colored via
// lipgloss so the terminal, not this code, decides the final look on
// non-truecolor hosts.
func (i backendItem) Title() string {
	mark := unavailableMark
	if i.status.Available {
		mark = availableMark
	}
	return fmt.Sprintf("%s %s", mark, i.status.Name)
}

// Description renders the one-line Reason plus the verbose block when the
// user has toggled `?`. Verbose lines are indented so the eye can scan the
// non-verbose column unchanged.
func (i backendItem) Description() string {
	return reasonStyle.Render(i.status.Reason)
}

func (i backendItem) FilterValue() string { return i.status.Name }

// selectorModel holds the Bubble Tea state: the list of backends, whether
// verbose has been toggled inline, the terminal choice, and a cancelled
// flag so Update can distinguish "user hit enter on empty list" from
// "user hit Ctrl+C".
type selectorModel struct {
	list      list.Model
	statuses  []BackendStatus
	verbose   bool
	choice    *backendChoice
	cancelled bool
}

// Init is required by tea.Model. There is no async work to kick off.
func (m selectorModel) Init() tea.Cmd { return nil }

// Update handles the two message classes we care about: window resize
// (propagate to the list) and key press (navigation / select / quit /
// verbose toggle). Everything else is forwarded to the list.
func (m selectorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		desired := compactHeight(len(m.statuses), m.verbose, m.statuses)
		if desired > msg.Height-v {
			desired = msg.Height - v
		}
		if desired < 1 {
			desired = 1
		}
		m.list.SetSize(msg.Width-h, desired)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "?":
			m.verbose = !m.verbose
			m.list.SetItems(m.buildItems())
			return m, nil
		case "enter":
			if item, ok := m.list.SelectedItem().(backendItem); ok {
				m.choice = &backendChoice{
					Kind:    item.status.Kind,
					Verbose: m.verbose,
				}
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the list plus a footer describing the verbose toggle state.
// The verbose block is appended per-item in buildItems, keeping the list
// itself layout-independent.
func (m selectorModel) View() string {
	verboseHint := "?: show details"
	if m.verbose {
		verboseHint = "?: hide details"
	}
	footer := reasonStyle.Render(fmt.Sprintf("↑/↓ or k/j: navigate • enter: select • %s • q/esc: cancel", verboseHint))
	return m.list.View() + "\n" + footer + "\n"
}

// buildItems constructs the list.Item slice. When verbose is on, each
// backend's Verbose diagnostic lines are folded into its Description so the
// user sees them without leaving the TUI.
func (m selectorModel) buildItems() []list.Item {
	items := make([]list.Item, 0, len(m.statuses))
	for _, s := range m.statuses {
		items = append(items, backendItem{status: s})
	}
	if !m.verbose {
		return items
	}
	// Reproject with verbose descriptions. We build a distinct wrapper so
	// the toggle stays a pure state change on the model.
	verboseItems := make([]list.Item, 0, len(m.statuses))
	for _, s := range m.statuses {
		desc := reasonStyle.Render(s.Reason)
		for _, line := range s.Verbose {
			desc += "\n  " + reasonStyle.Render(line)
		}
		verboseItems = append(verboseItems, verboseBackendItem{
			base: backendItem{status: s},
			desc: desc,
		})
	}
	return verboseItems
}

// verboseBackendItem carries the pre-rendered verbose description. It
// preserves the base item's Title/FilterValue so navigation logic keeps
// working after the toggle.
type verboseBackendItem struct {
	base backendItem
	desc string
}

func (i verboseBackendItem) Title() string       { return i.base.Title() }
func (i verboseBackendItem) Description() string { return i.desc }
func (i verboseBackendItem) FilterValue() string { return i.base.FilterValue() }

// nonTTYHint is the exact stderr message shown when the selector cannot
// run. It names every non-interactive escape hatch so the user does not
// have to consult the docs to unblock themselves.
const nonTTYHint = `Not a TTY. To select a backend non-interactively:
  obscuro auth store --backend=keychain
  obscuro auth store --backend=file
Or provide the password directly:
  OBSCURO_PASSWORD=... obscuro auth store
  obscuro auth store --password-file /path/to/pw
`

// compactHeight computes the minimum list height needed to show all items
// without padding the selector to the full terminal viewport.
// Chrome = title(1) + pagination(1) + footer-row(1) + vertical-margin(2) = 5.
// Each item costs itemHeight(2) rows, plus itemSpacing(1) between items
// (NOT after the last item), so for n items: n*itemHeight + max(0,n-1)*itemSpacing.
// In verbose mode each item additionally costs len(s.Verbose) rows.
// Result is clamped to a minimum of chrome(5) so chrome is never clipped.
func compactHeight(n int, verbose bool, statuses []BackendStatus) int {
	const (
		chrome      = 5 // title + pagination + footer + 2 vertical-margin
		itemHeight  = 2 // DefaultDelegate height when ShowDescription=true
		itemSpacing = 1 // DefaultDelegate spacing (between items, not after last)
	)
	if n <= 0 {
		return chrome
	}
	h := chrome + n*itemHeight + (n-1)*itemSpacing
	if verbose {
		for i := 0; i < n && i < len(statuses); i++ {
			h += len(statuses[i].Verbose)
		}
	}
	return h
}

// runBackendSelector is the production entry point. It refuses to run
// without a TTY (returning ErrNonInteractive) and otherwise drives a
// bubbles/list model until the user selects a backend or cancels.
func runBackendSelector(statuses []BackendStatus, verbose bool) (backendChoice, error) {
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Fprint(os.Stderr, nonTTYHint)
		return backendChoice{}, ErrNonInteractive
	}

	delegate := list.NewDefaultDelegate()
	m := selectorModel{
		statuses: statuses,
		verbose:  verbose,
	}
	// Width/height are seeded here; the first WindowSizeMsg will resize
	// the list to the real terminal viewport.
	m.list = list.New(m.buildItems(), delegate, 60, compactHeight(len(statuses), verbose, statuses))
	m.list.Title = "Select password backend"
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(false)
	m.list.SetShowHelp(false)

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return backendChoice{}, fmt.Errorf("running backend selector: %w", err)
	}

	fm, ok := final.(selectorModel)
	if !ok {
		return backendChoice{}, fmt.Errorf("unexpected final model type %T", final)
	}
	if fm.cancelled || fm.choice == nil {
		return backendChoice{}, ErrCancelled
	}
	return *fm.choice, nil
}
