package cmd

import (
	"fmt"
	"os"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-isatty"
)

// ImportChoice represents the user's selection in the import conflict prompt.
type ImportChoice string

const (
	// ImportChoiceNewOnly imports only new secrets, skipping pre-existing ones.
	ImportChoiceNewOnly ImportChoice = "new-only"
	// ImportChoiceOverwrite imports all secrets, overwriting pre-existing ones.
	ImportChoiceOverwrite ImportChoice = "overwrite"
	// ImportChoiceCancel aborts the import without making any changes.
	ImportChoiceCancel ImportChoice = "cancel"
)

// runImportChoiceFn is the test seam mirroring runBackendSelectorFn.
// Tests replace this with a stub; production wiring points to runImportChoice.
var runImportChoiceFn = runImportChoice

// importItem adapts an ImportChoice into a bubbles/list.Item.
type importItem struct {
	title  string
	choice ImportChoice
}

func (i importItem) Title() string       { return i.title }
func (i importItem) Description() string { return "" }
func (i importItem) FilterValue() string { return i.title }

// importModel holds the Bubble Tea state for the import choice prompt.
type importModel struct {
	list      list.Model
	choice    *ImportChoice
	cancelled bool
}

// Init is required by tea.Model. No async work to kick off.
func (m importModel) Init() tea.Cmd { return nil }

// Update handles key presses and window resizes.
func (m importModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		h, v := lipgloss.NewStyle().Margin(1, 2).GetFrameSize()
		height := msg.Height - v
		if height < 1 {
			height = 1
		}
		m.list.SetSize(msg.Width-h, height)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			m.cancelled = true
			return m, tea.Quit
		case "enter":
			if item, ok := m.list.SelectedItem().(importItem); ok {
				m.choice = &item.choice
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

// View renders the list and footer hint.
func (m importModel) View() string {
	footer := reasonStyle.Render("↑/↓ or k/j: navigate • enter: select • q/esc: cancel")
	return m.list.View() + "\n" + footer + "\n"
}

const importNonTTYHint = `Not a TTY. To import non-interactively, use:
  obscuro import FILE --on-conflict=skip
  obscuro import FILE --on-conflict=overwrite
  obscuro import FILE --on-conflict=fail
Or run in an interactive terminal.
`

// runImportChoice presents the interactive import conflict prompt.
// Returns ErrNonInteractive when stdin is not a TTY, ErrCancelled when the
// user quits without selecting.
func runImportChoice(newCount, existingCount int) (ImportChoice, error) {
	if !isatty.IsTerminal(os.Stdin.Fd()) {
		fmt.Fprint(os.Stderr, importNonTTYHint)
		return ImportChoiceCancel, ErrNonInteractive
	}

	var items []list.Item
	if existingCount > 0 {
		items = []list.Item{
			importItem{title: "Import new secrets only", choice: ImportChoiceNewOnly},
			importItem{title: "Import new and overwrite existing", choice: ImportChoiceOverwrite},
			importItem{title: "Cancel", choice: ImportChoiceCancel},
		}
	} else {
		items = []list.Item{
			importItem{title: fmt.Sprintf("Import %d new secret(s)", newCount), choice: ImportChoiceNewOnly},
			importItem{title: "Cancel", choice: ImportChoiceCancel},
		}
	}

	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false
	n := len(items)
	m := importModel{}
	m.list = list.New(items, delegate, 60, compactHeight(n, false, nil))
	m.list.Title = "How should conflicts be handled?"
	m.list.SetShowStatusBar(false)
	m.list.SetFilteringEnabled(false)
	m.list.SetShowHelp(false)

	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return ImportChoiceCancel, fmt.Errorf("running import choice: %w", err)
	}

	fm, ok := final.(importModel)
	if !ok {
		return ImportChoiceCancel, fmt.Errorf("unexpected final model type %T", final)
	}
	if fm.cancelled || fm.choice == nil {
		return ImportChoiceCancel, ErrCancelled
	}
	return *fm.choice, nil
}
