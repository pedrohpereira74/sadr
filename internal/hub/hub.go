package hub

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sahilm/fuzzy"

	"github.com/pedrohpereira74/sadr/internal/search"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
)

type Mode int

const (
	ModeSearch Mode = iota
	ModeDelete
	ModeExport
)

type ExportMode int

const (
	ExportFull ExportMode = iota
	ExportAdr
	ExportSnippet
)

func (e ExportMode) String() string {
	switch e {
	case ExportFull:
		return "full"
	case ExportAdr:
		return "adr"
	case ExportSnippet:
		return "snippet"
	default:
		return "unknown"
	}
}

type hubState int

const (
	stateLoading hubState = iota
	stateReady
	stateConfirmDelete
	stateExportOptions
)

type Options struct {
	Mode       Mode
	RecordDirs []string
	ExportsDir string
	UserFilter string
	OnExport   func(entry storage.RecordEntry, mode ExportMode) error
	FindEditor func() string
	OpenEditor func(editor, path string) error
}

type entriesLoadedMsg []storage.RecordEntry
type actionResultMsg struct {
	text  string
	isErr bool
}
type clearFeedbackMsg struct{}

type Model struct {
	opts  Options
	state hubState

	allEntries   []storage.RecordEntry
	filtered     []storage.RecordEntry
	deepContexts []string
	cursor       int
	offset       int

	input textinput.Model
	deep  bool

	pendingEntry *storage.RecordEntry
	exportMode   ExportMode

	feedback    string
	feedbackErr bool

	width  int
	height int
}

func New(opts Options) *Model {
	ti := textinput.New()
	ti.Placeholder = "search records..."
	ti.Prompt = "> "
	ti.CharLimit = 200
	ti.Focus()

	return &Model{
		opts:  opts,
		state: stateLoading,
		input: ti,
	}
}

func Run(opts Options) error {
	m := New(opts)
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		loadEntriesCmd(m.opts.RecordDirs, m.opts.UserFilter),
	)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = max(10, msg.Width-6)
		return m, nil

	case entriesLoadedMsg:
		m.allEntries = msg
		m.state = stateReady
		m.applyFilter()
		return m, nil

	case clearFeedbackMsg:
		m.feedback = ""
		m.feedbackErr = false
		return m, nil

	case actionResultMsg:
		m.feedback = msg.text
		m.feedbackErr = msg.isErr
		m.state = stateReady
		m.input.Focus()
		clearCmd := tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearFeedbackMsg{} })
		return m, tea.Batch(textinput.Blink, clearCmd)

	case tea.KeyMsg:
		switch m.state {
		case stateLoading:
			if msg.Type == tea.KeyCtrlC {
				return m, tea.Quit
			}
		case stateReady:
			return m.handleReadyKey(msg)
		case stateConfirmDelete:
			return m.handleConfirmDeleteKey(msg)
		case stateExportOptions:
			return m.handleExportOptionsKey(msg)
		}
	}

	return m, nil
}

func (m *Model) handleReadyKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	m.feedback = ""

	if msg.Type == tea.KeyCtrlC {
		return m, tea.Quit
	}

	if msg.Type == tea.KeyCtrlD {
		if len(m.filtered) > 0 {
			m.pendingEntry = new(m.filtered[m.cursor])
			m.state = stateConfirmDelete
			m.input.Blur()
		}
		return m, nil
	}

	if msg.Type == tea.KeyCtrlE {
		if len(m.filtered) > 0 {
			m.pendingEntry = new(m.filtered[m.cursor])
			m.exportMode = ExportFull
			m.state = stateExportOptions
			m.input.Blur()
		}
		return m, nil
	}

	if msg.Type == tea.KeyUp {
		if m.cursor > 0 {
			m.cursor--
			if m.cursor < m.offset {
				m.offset = m.cursor
			}
		}
		return m, nil
	}

	if msg.Type == tea.KeyDown {
		if m.cursor < len(m.filtered)-1 {
			m.cursor++
			visible := m.visibleEntries()
			if m.cursor >= m.offset+visible {
				m.offset = m.cursor - visible + 1
			}
		}
		return m, nil
	}

	if msg.Type == tea.KeyTab {
		m.deep = !m.deep
		m.applyFilter()
		return m, nil
	}

	if m.input.Focused() {
		switch msg.Type {
		case tea.KeyEnter:
			if len(m.filtered) > 0 {
				return m.triggerAction(m.filtered[m.cursor])
			}
			return m, nil
		}
		prevVal := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != prevVal {
			m.applyFilter()
			m.cursor = 0
			m.offset = 0
		}
		return m, cmd
	}

	switch msg.Type {
	case tea.KeyEnter:
		if len(m.filtered) > 0 {
			return m.triggerAction(m.filtered[m.cursor])
		}
		return m, nil

	case tea.KeyRunes:
		m.input.Focus()
		prevVal := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != prevVal {
			m.applyFilter()
			m.cursor = 0
			m.offset = 0
		}
		return m, tea.Batch(textinput.Blink, cmd)

	default:
		return m, nil
	}
}

func (m *Model) handleConfirmDeleteKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.state = stateReady
		m.pendingEntry = nil
		m.input.Focus()
		return m, textinput.Blink
	case tea.KeyRunes:
		switch msg.String() {
		case "y":
			return m.executeDelete()
		case "n":
			m.state = stateReady
			m.pendingEntry = nil
			m.input.Focus()
			return m, textinput.Blink
		default:
			return m, nil
		}
	default:
		return m, nil
	}
}

func (m *Model) handleExportOptionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		return m, tea.Quit
	case tea.KeyEsc:
		m.state = stateReady
		m.pendingEntry = nil
		m.input.Focus()
		return m, textinput.Blink
	case tea.KeyEnter:
		return m.executeExport()
	case tea.KeyTab:
		m.exportMode = (m.exportMode + 1) % 3
		return m, nil
	default:
		return m, nil
	}
}

func (m *Model) triggerAction(entry storage.RecordEntry) (tea.Model, tea.Cmd) {
	switch m.opts.Mode {
	case ModeDelete:
		m.pendingEntry = &entry
		m.state = stateConfirmDelete
		return m, nil
	case ModeExport:
		m.pendingEntry = &entry
		m.exportMode = ExportFull
		m.state = stateExportOptions
		return m, nil
	default:
		return m.launchEditor(entry)
	}
}

func (m *Model) launchEditor(entry storage.RecordEntry) (tea.Model, tea.Cmd) {
	if m.opts.FindEditor == nil {
		m.feedback = "no editor configured."
		m.feedbackErr = true
		return m, nil
	}
	editor := m.opts.FindEditor()
	if editor == "" {
		m.feedback = "no editor found. set $EDITOR or add 'editor' to ~/.sadr/global-config.yaml"
		m.feedbackErr = true
		return m, nil
	}

	parts := strings.Fields(editor)
	args := append(parts[1:], entry.Path)
	c := exec.Command(parts[0], args...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	title := entry.Record.Title
	return m, tea.ExecProcess(c, func(err error) tea.Msg {
		if err != nil {
			return actionResultMsg{text: fmt.Sprintf("editor exited with error: %v", err), isErr: true}
		}
		return actionResultMsg{text: fmt.Sprintf("%s updated.", title), isErr: false}
	})
}

func (m *Model) executeDelete() (tea.Model, tea.Cmd) {
	if m.pendingEntry == nil {
		m.state = stateReady
		return m, nil
	}
	entry := *m.pendingEntry
	m.pendingEntry = nil

	s := storage.NewStorage(filepath.Dir(entry.Path))
	err := s.DeleteRecord(entry.Path)

	newAll := make([]storage.RecordEntry, 0, len(m.allEntries))
	for _, e := range m.allEntries {
		if e.Path != entry.Path {
			newAll = append(newAll, e)
		}
	}
	m.allEntries = newAll
	m.applyFilter()
	if m.cursor >= len(m.filtered) && m.cursor > 0 {
		m.cursor--
	}
	m.state = stateReady
	m.input.Focus()

	if err != nil {
		m.feedback = fmt.Sprintf("failed to delete: %v", err)
		m.feedbackErr = true
	} else {
		m.feedback = fmt.Sprintf("%s deleted.", entry.Record.Title)
		m.feedbackErr = false
	}
	return m, textinput.Blink
}

func (m *Model) executeExport() (tea.Model, tea.Cmd) {
	if m.pendingEntry == nil || m.opts.OnExport == nil {
		m.state = stateReady
		return m, nil
	}
	entry := *m.pendingEntry
	mode := m.exportMode
	m.pendingEntry = nil

	onExport := m.opts.OnExport
	title := entry.Record.Title

	return m, func() tea.Msg {
		err := onExport(entry, mode)
		if err != nil {
			return actionResultMsg{text: fmt.Sprintf("failed to export: %v", err), isErr: true}
		}
		return actionResultMsg{text: fmt.Sprintf("%s exported.", title), isErr: false}
	}
}

func (m *Model) View() string {
	if m.state == stateLoading {
		return "\n  loading records...\n"
	}

	var b strings.Builder

	modeNames := map[Mode]string{
		ModeSearch: "search",
		ModeDelete: "delete",
		ModeExport: "export",
	}
	deepTag := ""
	if m.deep {
		deepTag = "  [deep]"
	}
	_, _ = fmt.Fprintf(&b, "\n  sadr — %s%s\n\n", modeNames[m.opts.Mode], deepTag)

	_, _ = fmt.Fprintf(&b, "  %s\n", m.input.View())

	sepWidth := max(1, m.width-4)
	_, _ = fmt.Fprintf(&b, "  %s\n\n", strings.Repeat("─", sepWidth))

	switch m.state {
	case stateConfirmDelete:
		b.WriteString(m.viewConfirmDelete())
	case stateExportOptions:
		b.WriteString(m.viewExportOptions())
	default:
		b.WriteString(m.viewList())
	}

	b.WriteString(m.viewFooter())

	return b.String()
}

func (m *Model) viewList() string {
	var b strings.Builder
	lh := m.listHeight()

	if len(m.filtered) == 0 {
		if m.input.Value() != "" {
			_, _ = fmt.Fprintf(&b, "  no results for %q.\n", m.input.Value())
		} else {
			b.WriteString("  no records found.\n")
		}
		for i := 1; i < lh; i++ {
			b.WriteByte('\n')
		}
		return b.String()
	}

	typeWidth := 8
	tagMaxWidth := 24
	titleWidth := max(10, m.width-2-2-typeWidth-2-tagMaxWidth-2)

	renderedLines := 0
	for i := m.offset; i < len(m.filtered) && renderedLines < lh; i++ {
		e := m.filtered[i]
		cur := "  "
		if i == m.cursor {
			cur = "> "
		}

		typeStr := fmt.Sprintf("%-*s", typeWidth, e.Record.Type)
		title := truncate(e.Record.Title, titleWidth)
		titleStr := fmt.Sprintf("%-*s", titleWidth, title)

		tags := strings.Join(e.Record.Tags, ", ")
		if tags == "" {
			tags = "—"
		}
		tags = truncate(tags, tagMaxWidth)

		_, _ = fmt.Fprintf(&b, "  %s%s  %s  %s\n", cur, typeStr, titleStr, tags)
		renderedLines++

		if m.deep && renderedLines < lh {
			ctx := ""
			if i < len(m.deepContexts) {
				ctx = m.deepContexts[i]
			}
			if ctx != "" {
				_, _ = fmt.Fprintf(&b, "       %s\n", ctx)
			} else {
				b.WriteByte('\n')
			}
			renderedLines++
		}
	}

	for renderedLines < lh {
		b.WriteByte('\n')
		renderedLines++
	}

	return b.String()
}

func (m *Model) viewConfirmDelete() string {
	var b strings.Builder
	lh := m.listHeight()

	if m.pendingEntry != nil {
		titleMax := max(10, m.width-20)
		title := truncate(m.pendingEntry.Record.Title, titleMax)
		_, _ = fmt.Fprintf(&b, "  delete \"%s\"?\n\n", title)
		for i := 2; i < lh; i++ {
			b.WriteByte('\n')
		}
	} else {
		for range lh {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func (m *Model) viewExportOptions() string {
	var b strings.Builder
	lh := m.listHeight()

	if m.pendingEntry != nil {
		titleMax := max(10, m.width-20)
		title := truncate(m.pendingEntry.Record.Title, titleMax)
		modeStr := m.exportMode.String()

		_, _ = fmt.Fprintf(&b, "  export \"%s\"\n\n", title)
		_, _ = fmt.Fprintf(&b, "  export mode: %s\n\n", modeStr)
		for i := 4; i < lh; i++ {
			b.WriteByte('\n')
		}
	} else {
		for range lh {
			b.WriteByte('\n')
		}
	}

	return b.String()
}

func (m *Model) viewFooter() string {
	if m.feedback != "" {
		sym := "✓"
		if m.feedbackErr {
			sym = "✗"
		}
		return fmt.Sprintf("  %s %s\n", sym, truncate(m.feedback, m.width-6))
	}

	if m.state == stateConfirmDelete {
		return ui.HintsBar("y yes", "n no", "ctrl+c quit")
	}

	if m.state == stateExportOptions {
		return ui.HintsBar("enter export", "tab cycle mode", "esc cancel", "ctrl+c quit")
	}

	enterHint := map[Mode]string{
		ModeSearch: "enter edit",
		ModeDelete: "enter delete",
		ModeExport: "enter export",
	}[m.opts.Mode]

	return ui.HintsBar(enterHint, "ctrl+d delete", "ctrl+e export", "tab deep", "ctrl+c quit")
}

func (m *Model) applyFilter() {
	query := m.input.Value()
	if query == "" {
		m.filtered = make([]storage.RecordEntry, len(m.allEntries))
		copy(m.filtered, m.allEntries)
		m.deepContexts = nil
		return
	}

	titles := make([]string, len(m.allEntries))
	for i, e := range m.allEntries {
		titles[i] = e.Record.Title
	}
	fuzzyResults := fuzzy.Find(query, titles)
	seen := make(map[int]bool, len(fuzzyResults))
	m.filtered = m.filtered[:0]
	for _, r := range fuzzyResults {
		seen[r.Index] = true
		m.filtered = append(m.filtered, m.allEntries[r.Index])
	}
	for i, e := range m.allEntries {
		if seen[i] {
			continue
		}
		if search.MatchesTags(e.Record, query) || (m.deep && search.MatchesDeep(e.Record, query)) {
			m.filtered = append(m.filtered, e)
		}
	}

	if m.deep {
		m.deepContexts = make([]string, len(m.filtered))
		for i, e := range m.filtered {
			m.deepContexts[i] = search.DeepContext(e.Record, query)
		}
	} else {
		m.deepContexts = nil
	}
}

func (m *Model) listHeight() int {
	h := m.height - 8
	if h < 3 {
		return 3
	}
	return h
}

func (m *Model) visibleEntries() int {
	lh := m.listHeight()
	if m.deep {
		if lh/2 < 2 {
			return 2
		}
		return lh / 2
	}
	return lh
}

func truncate(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return string(runes[:maxLen])
	}
	return string(runes[:maxLen-1]) + "…"
}

func loadEntriesCmd(dirs []string, userFilter string) tea.Cmd {
	return func() tea.Msg {
		type fileTask struct {
			name string
			path string
		}

		var tasks []fileTask
		for _, dir := range dirs {
			entries, err := os.ReadDir(dir)
			if err != nil {
				continue
			}
			for _, e := range entries {
				if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
					tasks = append(tasks, fileTask{
						name: e.Name(),
						path: filepath.Join(dir, e.Name()),
					})
				}
			}
		}

		if len(tasks) == 0 {
			return entriesLoadedMsg(nil)
		}

		numWorkers := min(8, runtime.NumCPU(), len(tasks))

		taskCh := make(chan fileTask, len(tasks))
		resultCh := make(chan storage.RecordEntry, len(tasks))

		var wg sync.WaitGroup
		for range numWorkers {
			wg.Go(func() {
				for t := range taskCh {
					r, err := storage.LoadRecord(t.path)
					if err != nil {
						continue
					}
					if userFilter != "" && r.Author != userFilter {
						continue
					}
					resultCh <- storage.RecordEntry{
						Record: r,
						FileID: storage.ParseFileID(t.name),
						Path:   t.path,
						Author: r.Author,
					}
				}
			})
		}

		for _, t := range tasks {
			taskCh <- t
		}
		close(taskCh)

		go func() {
			wg.Wait()
			close(resultCh)
		}()

		var results []storage.RecordEntry
		for e := range resultCh {
			results = append(results, e)
		}

		sort.Slice(results, func(i, j int) bool {
			if results[i].Author != results[j].Author {
				return results[i].Author < results[j].Author
			}
			return results[i].FileID < results[j].FileID
		})

		return entriesLoadedMsg(results)
	}
}
