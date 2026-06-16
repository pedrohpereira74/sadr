package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/pedrohpereira74/sadr/internal/ask"
	"github.com/pedrohpereira74/sadr/internal/config"
	"github.com/pedrohpereira74/sadr/internal/discover"
	jiraenricher "github.com/pedrohpereira74/sadr/internal/enricher/jira"
	jiraclient "github.com/pedrohpereira74/sadr/internal/jira"
	"github.com/pedrohpereira74/sadr/internal/storage"
	"github.com/pedrohpereira74/sadr/internal/ui"
)

func DefaultPersonas() []ask.Persona {
	return []ask.Persona{
		{
			Name:        "Tech Lead",
			Instruction: "Analyze decisions strictly through an architecture lens: technical debt, design inconsistencies, coupling, and maintainability. If a finding is not directly related to code architecture or system design, do not include it.",
		},
		{
			Name:        "DBA",
			Instruction: "Analyze decisions strictly through a database lens: normalization, query patterns, indexing, data integrity, and persistence. If a finding is not directly related to data modeling, storage, or data access, do not include it.",
		},
		{
			Name:        "QA Engineer",
			Instruction: "Analyze decisions strictly through a quality assurance lens: testability, edge cases, regression risks, and test coverage. If a finding is not directly related to testing or quality verification, do not include it.",
		},
		{
			Name:        "Security Analyst",
			Instruction: "Analyze decisions strictly through a security lens: vulnerabilities, authentication gaps, data exposure, and compliance risks. If a finding is not directly related to security, do not include it.",
		},
		{
			Name:        "DevOps Engineer",
			Instruction: "Analyze decisions strictly through a DevOps lens: deployment risks, infrastructure concerns, CI/CD impacts, and operational reliability. If a finding is not directly related to deployment or operations, do not include it.",
		},
		{
			Name:        "UX Designer",
			Instruction: "Analyze decisions strictly through a UX and product design lens: user flows, interface consistency, accessibility, and the impact of technical decisions on the end-user experience. If a finding is not directly related to usability or design, do not include it.",
		},
	}
}

func PersonaSlug(name string) string {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")
	return s
}

func resolvePersonaByName(name string) ask.Persona {
	lower := strings.ToLower(name)
	for _, p := range DefaultPersonas() {
		if strings.ToLower(p.Name) == lower {
			return p
		}
	}
	return ask.Persona{
		Name:        name,
		Instruction: fmt.Sprintf("Analyze the architecture decisions from the perspective of a %s.", name),
	}
}

func selectPersona() ask.Persona {
	personas := DefaultPersonas()
	options := make([]selectOption, 0, len(personas)+1)
	for _, p := range personas {
		options = append(options, selectOption{Label: p.Name, Value: p.Name})
	}
	options = append(options, selectOption{Label: "custom...", Value: "custom"})

	chosen := runSelect("select a persona:", options)
	if chosen == "" {
		return ask.Persona{}
	}

	if chosen == "custom" {
		desc := runTextarea("describe your custom persona:", "e.g. a frontend performance expert focused on bundle size and rendering...")
		if desc == "" {
			return ask.Persona{}
		}
		return ask.Persona{
			Name:        "Custom",
			Instruction: desc,
		}
	}

	for _, p := range personas {
		if p.Name == chosen {
			return p
		}
	}
	return ask.Persona{}
}

func warnIfJiraNotConfiguredForProject(projectJiraURL string, hasJiraField bool) {
	cfg := loadGlobalConfig()
	j := cfg.Jira
	if j.DisableProjectWarning {
		return
	}
	hasCredentials := j.Username != "" || j.Token != "" || j.TokenEnv != "" || j.ConsumerKey != ""
	if !hasCredentials {
		return
	}
	if projectJiraURL == "" && !hasJiraField {
		ui.Warning(os.Stderr, "jira credentials configured but this project has no jira setup.")
		ui.Warning(os.Stderr, "add 'jira.url' and a field of type 'jira' to your .sadr/configs/*.yaml to enable it.")
		ui.Warning(os.Stderr, "to suppress this warning: sadr config --disable-jira-warning")
	}
}

func loadProjectJiraURL(configsDir string) string {
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		cfg, err := config.LoadFromFile(filepath.Join(configsDir, e.Name()))
		if err != nil {
			continue
		}
		if cfg.Jira.URL != "" {
			return cfg.Jira.URL
		}
	}
	return ""
}

func loadJiraEnricher(projectURL string) *jiraenricher.Enricher {
	cfg := loadGlobalConfig()
	j := cfg.Jira
	return jiraenricher.New(projectURL, jiraclient.ClientConfig{
		Username:          j.Username,
		Password:          j.Password,
		PasswordEnv:       j.PasswordEnv,
		Token:             j.Token,
		TokenEnv:          j.TokenEnv,
		ConsumerKey:       j.ConsumerKey,
		PrivateKeyPath:    j.PrivateKeyPath,
		AccessToken:       j.AccessToken,
		AccessTokenSecret: j.AccessTokenSecret,
	})
}

func loadGlobalConfig() config.GlobalConfig {
	home, err := os.UserHomeDir()
	if err != nil {
		return config.GlobalConfig{}
	}
	cfg, err := config.LoadGlobalFromFile(filepath.Join(home, ".sadr", "global-config.yaml"))
	if err != nil {
		return config.GlobalConfig{}
	}
	return cfg
}

func loadAIConfig() (provider, apiKey, model string) {
	cfg := loadGlobalConfig()
	apiKey = cfg.AI.APIKey
	if apiKey == "" && cfg.AI.APIKeyEnv != "" {
		apiKey = os.Getenv(cfg.AI.APIKeyEnv)
	}
	return cfg.AI.Provider, apiKey, cfg.AI.Model
}

func loadLanguageConfig() string {
	cfg := loadGlobalConfig()
	if cfg.Language == "" {
		return "English"
	}
	return cfg.Language
}

func parseID(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0, fmt.Errorf("invalid id %q: must be a positive number", raw)
	}
	return n, nil
}

func loadUsername() string {
	return loadGlobalConfig().Username
}

func resolveRecordDirs(global bool, paths discover.SadrPaths) []string {
	if global {
		return []string{paths.Records}
	}
	return allUserRecordsDirs(paths.Root)
}

func allUserRecordsDirs(sadrRoot string) []string {
	entries, err := os.ReadDir(sadrRoot)
	if err != nil {
		return nil
	}
	var dirs []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		recordsDir := filepath.Join(sadrRoot, e.Name(), "records")
		if _, err := os.Stat(recordsDir); err == nil {
			dirs = append(dirs, recordsDir)
		}
	}
	return dirs
}

func listAllRecordEntries(sadrRoot string) ([]storage.RecordEntry, error) {
	dirs := allUserRecordsDirs(sadrRoot)
	var all []storage.RecordEntry
	for _, dir := range dirs {
		s := storage.NewStorage(dir)
		entries, err := s.ListRecordEntries()
		if err != nil {
			ui.Warning(os.Stderr, fmt.Sprintf("skipping %s: %v", dir, err))
			continue
		}
		all = append(all, entries...)
	}
	return all, nil
}

func pickConfigFile(configsDir string) (string, error) {
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("configs directory not found: %q", configsDir)
		}
		return "", fmt.Errorf("failed to read configs directory %q: %w", configsDir, err)
	}
	var configs []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			configs = append(configs, e.Name())
		}
	}
	if len(configs) == 0 {
		return "", fmt.Errorf("no config files found in %q", configsDir)
	}
	if len(configs) == 1 {
		return filepath.Join(configsDir, configs[0]), nil
	}
	options := make([]selectOption, 0, len(configs))
	for _, f := range configs {
		name := configDisplayName(f)
		options = append(options, selectOption{Label: name, Value: f})
	}
	chosen := runSelect("which config?", options)
	if chosen == "" {
		return "", fmt.Errorf("cancelled")
	}
	return filepath.Join(configsDir, chosen), nil
}

func parseUserID(raw string) (username string, id int, err error) {
	if raw == "" {
		return "", 0, nil
	}
	if before, after, found := strings.Cut(raw, "/"); found {
		username = before
		id, err = strconv.Atoi(after)
	} else {
		id, err = strconv.Atoi(raw)
	}
	if err != nil || id < 0 {
		return "", 0, fmt.Errorf("invalid id %q: must be number or name/number", raw)
	}
	return
}

type selectOption struct {
	Label string
	Value string
}

type selectModel struct {
	prompt  string
	options []selectOption
	cursor  int
	chosen  string
	done    bool
}

func (m selectModel) Init() tea.Cmd { return nil }

func (m selectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyUp:
			if m.cursor > 0 {
				m.cursor--
			}
		case tea.KeyDown:
			if m.cursor < len(m.options)-1 {
				m.cursor++
			}
		case tea.KeyEnter:
			m.chosen = m.options[m.cursor].Value
			m.done = true
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m selectModel) View() string {
	if m.done {
		return ""
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\n%s\n\n", m.prompt)
	for i, opt := range m.options {
		cursor := "  "
		if i == m.cursor {
			cursor = "> "
		}
		fmt.Fprintf(&b, "  %s%s\n", cursor, opt.Label)
	}
	return b.String()
}

func runSelect(prompt string, options []selectOption) string {
	m := selectModel{prompt: prompt, options: options}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}
	return finalModel.(selectModel).chosen
}

type textareaModel struct {
	prompt   string
	textarea textarea.Model
	result   string
	done     bool
}

func (m textareaModel) Init() tea.Cmd { return textarea.Blink }

func (m textareaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc:
			return m, tea.Quit
		case tea.KeyEnter:
			if !msg.Alt {
				m.result = m.textarea.Value()
				m.done = true
				return m, tea.Quit
			}
		}
	}
	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

func (m textareaModel) View() string {
	if m.done {
		return ""
	}
	return fmt.Sprintf("\n%s\n\n%s\n\n%s", m.prompt, m.textarea.View(), ui.HintsBar("enter confirm", "esc cancel", "ctrl+c quit"))
}

func runTextarea(prompt string, placeholder string) string {
	ta := textarea.New()
	ta.Placeholder = placeholder
	ta.Focus()
	ta.SetWidth(70)
	ta.SetHeight(3)
	ta.ShowLineNumbers = false

	m := textareaModel{prompt: prompt, textarea: ta}
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return ""
	}
	return finalModel.(textareaModel).result
}

func confirmPromptImpl(message string) bool {
	chosen := runSelect(message, []selectOption{
		{Label: "yes, proceed", Value: "yes"},
		{Label: "no, cancel", Value: "no"},
	})
	return chosen == "yes"
}

func promptGlobalFallbackImpl() string {
	return runSelect("no local sadr project found. fallback to global at ~/.sadr?", []selectOption{
		{Label: "yes, use global", Value: "yes"},
		{Label: "no, cancel", Value: "no"},
	})
}

func handleGlobalFallback(paths discover.SadrPaths) error {
	if !paths.IsGlobal {
		return nil
	}
	chosen := fallbackPrompter()
	switch chosen {
	case "yes":
		return nil
	default:
		ui.Success(os.Stderr, "run 'sadr init' in your project to initialize a local sadr project.")
		return fmt.Errorf("cancelled")
	}
}

func findEditor() string {
	if editor := os.Getenv("VISUAL"); editor != "" {
		return editor
	}
	if editor := os.Getenv("EDITOR"); editor != "" {
		return editor
	}
	if home, err := os.UserHomeDir(); err == nil {
		globalConfigPath := filepath.Join(home, ".sadr", "global-config.yaml")
		if cfg, err := config.LoadGlobalFromFile(globalConfigPath); err == nil && cfg.Editor != "" {
			return cfg.Editor
		}
	}
	if runtime.GOOS == "windows" {
		for _, fallback := range []string{"notepad", "code"} {
			if _, err := exec.LookPath(fallback); err == nil {
				if fallback == "code" {
					return "code --wait"
				}
				return fallback
			}
		}
	} else {
		for _, fallback := range []string{"vim", "nano", "vi"} {
			if _, err := exec.LookPath(fallback); err == nil {
				return fallback
			}
		}
	}
	return ""
}

func openEditorImpl(editor string, path string) error {
	parts := strings.Fields(editor)
	var c *exec.Cmd
	if len(parts) > 1 {
		c = exec.Command(parts[0], append(parts[1:], path)...)
	} else {
		c = exec.Command(editor, path)
	}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}
func configDisplayName(filename string) string {
	name := strings.TrimSuffix(filename, ".yaml")
	if name == "default-config" {
		return "default"
	}
	return name
}

func configFilename(name string) string {
	if name == "default" {
		return "default-config.yaml"
	}
	return name + ".yaml"
}

func resolvePaths(global bool) (discover.SadrPaths, error) {
	if global {
		home, err := os.UserHomeDir()
		if err != nil {
			return discover.SadrPaths{}, fmt.Errorf("could not find home directory: %v", err)
		}
		root := filepath.Join(home, ".sadr")
		if _, err := os.Stat(root); os.IsNotExist(err) {
			return discover.SadrPaths{}, fmt.Errorf("global storage not found. run 'sadr config --global' first")
		}
		return discover.SadrPaths{
			Root:       root,
			Records:    filepath.Join(root, "records"),
			Exports:    filepath.Join(root, "exports"),
			Answers:    filepath.Join(root, "answers"),
			ConfigsDir: filepath.Join(root, "configs"),
		}, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return discover.SadrPaths{}, fmt.Errorf("could not get working directory: %v", err)
	}
	paths, err := discover.FindSadrDir(dir)
	if err != nil {
		return paths, err
	}
	if err := handleGlobalFallback(paths); err != nil {
		return discover.SadrPaths{}, err
	}

	if !paths.IsGlobal {
		username := loadUsername()
		if username == "" {
			return discover.SadrPaths{}, fmt.Errorf("no user configured, run 'sadr init --global' first")
		}
		paths.Username = username
		paths.Records = filepath.Join(paths.Root, username, "records")
		paths.Exports = filepath.Join(paths.Root, username, "exports")
		paths.Answers = filepath.Join(paths.Root, username, "answers")
	} else {
		paths.Answers = filepath.Join(paths.Root, "answers")
	}

	return paths, nil
}

func truncateTitle(title, query string, maxLen int) string {
	runes := []rune(title)
	if len(runes) <= maxLen {
		return title
	}
	byteIdx := strings.Index(strings.ToLower(title), strings.ToLower(query))
	if byteIdx < 0 {
		return string(runes[:maxLen-3]) + "..."
	}
	const dots = 3
	window := maxLen - dots*2
	queryRunes := []rune(query)
	half := (window - len(queryRunes)) / 2
	runeIdx := utf8.RuneCountInString(title[:byteIdx])
	start := max(runeIdx-half, 0)
	end := start + window
	if end > len(runes) {
		end = len(runes)
		start = max(end-window, 0)
	}
	result := string(runes[start:end])
	if start > 0 {
		result = "..." + result
	}
	if end < len(runes) {
		result += "..."
	}
	return result
}
