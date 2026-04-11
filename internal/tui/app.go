package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/arjunmyanger/livepulse/internal/alert"
	"github.com/arjunmyanger/livepulse/internal/monitor"
	"github.com/arjunmyanger/livepulse/internal/store"
	"github.com/arjunmyanger/livepulse/internal/tui/components"
)

// view represents the current active view in the TUI.
type view int

const (
	viewDashboard view = iota
	viewDetail
	viewSetup
	viewHelp
)

// maxResultsPerMonitor caps stored check results per monitor.
const maxResultsPerMonitor = 100

// checkResultMsg wraps a monitor.CheckResult as a tea.Msg.
type checkResultMsg struct {
	result monitor.CheckResult
}

// tickMsg is sent periodically for UI refresh.
type tickMsg time.Time

// App is the main bubbletea model that ties together all views, the engine,
// alerts, and persistence.
type App struct {
	// View state
	currentView view
	monitors    []MonitorView
	selected    int
	results     map[string][]monitor.CheckResult
	width       int
	height      int

	// Components
	statusBar      *components.StatusBar
	eventLog       *components.EventLog
	eventsExpanded bool

	// Engine
	engine    *monitor.Engine
	resultsCh chan monitor.CheckResult

	// Alerts
	alertMgr   *alert.Manager
	visualDisp *alert.VisualDispatcher
	desktopDisp *alert.DesktopDispatcher
	webhooks   map[string]*alert.WebhookDispatcher

	// Store
	store   *store.Store
	configs []store.MonitorConfig

	// Sub-views
	setupModel setupModel
	detailID      string
	editing       bool   // true when setup is editing an existing monitor
	pendingDelete string // monitor ID awaiting delete confirmation
}

// NewApp creates a new App with sensible defaults.
func NewApp(storePath string) *App {
	resultsCh := make(chan monitor.CheckResult, 64)
	return &App{
		currentView: viewDashboard,
		results:     make(map[string][]monitor.CheckResult),
		statusBar:   components.NewStatusBar(80),
		eventLog:    components.NewEventLog(10),
		engine:      monitor.NewEngine(resultsCh),
		resultsCh:   resultsCh,
		alertMgr:    alert.NewManager(3, 2),
		visualDisp:  alert.NewVisualDispatcher(),
		desktopDisp: alert.NewDesktopDispatcher(),
		webhooks:    make(map[string]*alert.WebhookDispatcher),
		store:       store.New(storePath),
	}
}

// StartEngine starts the monitoring engine. Call this from main.go before
// tea.Program.Run so that goroutines are launched with a proper context.
func (a *App) StartEngine(ctx context.Context) {
	a.engine.Start(ctx)
}

// Init implements tea.Model. It loads saved monitors and returns initial commands.
func (a *App) Init() tea.Cmd {
	configs, err := a.store.Load()
	if err != nil {
		a.eventLog.Add("system", fmt.Sprintf("load error: %v", err), true)
		return tea.Batch(a.waitForResult(), tickCmd())
	}

	a.configs = configs
	for _, cfg := range configs {
		a.addMonitorFromConfig(cfg)
	}

	return tea.Batch(a.waitForResult(), tickCmd())
}

// Update implements tea.Model.
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case checkResultMsg:
		a.handleCheckResult(msg.result)
		return a, a.waitForResult()

	case tickMsg:
		return a, tickCmd()
	}

	// Delegate to view-specific update
	switch a.currentView {
	case viewDashboard:
		return a.updateDashboard(msg)
	case viewDetail:
		return a.updateDetail(msg)
	case viewSetup:
		return a.updateSetup(msg)
	case viewHelp:
		return a.updateHelp(msg)
	}

	return a, nil
}

// View implements tea.Model.
func (a *App) View() tea.View {
	switch a.currentView {
	case viewDashboard:
		return tea.NewView(renderDashboard(a.monitors, a.selected, a.statusBar, a.eventLog, a.eventsExpanded, a.width, a.height))
	case viewDetail:
		mv, results, spark := a.detailData()
		return tea.NewView(renderDetail(mv, results, spark, a.width))
	case viewSetup:
		return tea.NewView(a.setupModel.View())
	case viewHelp:
		return tea.NewView(renderHelp(a.width))
	}
	return tea.NewView("")
}

// --- View-specific update methods ---

func (a *App) updateDashboard(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return a, nil
	}

	key := keyMsg.Key()
	switch key.Text {
	case "q":
		a.clearPendingDelete()
		return a, tea.Quit
	case "j":
		a.clearPendingDelete()
		if a.selected < len(a.monitors)-1 {
			a.selected++
		}
	case "k":
		a.clearPendingDelete()
		if a.selected > 0 {
			a.selected--
		}
	case "a":
		a.clearPendingDelete()
		a.editing = false
		a.setupModel = newSetupModel(nil)
		a.currentView = viewSetup
	case "e":
		a.clearPendingDelete()
		if len(a.monitors) > 0 {
			cfg := a.configForID(a.monitors[a.selected].ID)
			if cfg != nil {
				a.editing = true
				a.detailID = cfg.ID
				a.setupModel = newSetupModel(cfg)
				a.currentView = viewSetup
			}
		}
	case "d":
		if len(a.monitors) > 0 {
			id := a.monitors[a.selected].ID
			if a.pendingDelete == id {
				a.deleteMonitor(id)
				a.pendingDelete = ""
				if a.selected >= len(a.monitors) && a.selected > 0 {
					a.selected--
				}
			} else {
				a.pendingDelete = id
				name := a.monitors[a.selected].Name
				a.eventLog.Add(name, "press d again to confirm delete", false)
			}
		}
	case "v":
		a.clearPendingDelete()
		a.eventsExpanded = !a.eventsExpanded
	case "?":
		a.clearPendingDelete()
		a.currentView = viewHelp
	default:
		if key.Code == tea.KeyEnter {
			a.clearPendingDelete()
			if len(a.monitors) > 0 {
				a.detailID = a.monitors[a.selected].ID
				a.currentView = viewDetail
			}
		}
	}

	return a, nil
}

func (a *App) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return a, nil
	}

	key := keyMsg.Key()
	switch {
	case key.Code == tea.KeyEscape:
		a.clearPendingDelete()
		a.currentView = viewDashboard
	case key.Text == "e":
		a.clearPendingDelete()
		cfg := a.configForID(a.detailID)
		if cfg != nil {
			a.editing = true
			a.setupModel = newSetupModel(cfg)
			a.currentView = viewSetup
		}
	case key.Text == "d":
		if a.pendingDelete == a.detailID {
			a.deleteMonitor(a.detailID)
			a.pendingDelete = ""
			a.currentView = viewDashboard
			if a.selected >= len(a.monitors) && a.selected > 0 {
				a.selected--
			}
		} else {
			a.pendingDelete = a.detailID
			name := ""
			for _, m := range a.monitors {
				if m.ID == a.detailID {
					name = m.Name
					break
				}
			}
			a.eventLog.Add(name, "press d again to confirm delete", false)
		}
	}

	return a, nil
}

func (a *App) updateSetup(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyPressMsg)

	// Check for escape to cancel
	if isKey && keyMsg.Key().Code == tea.KeyEscape {
		a.currentView = viewDashboard
		return a, nil
	}

	// Check for enter on confirm step to save
	if isKey && keyMsg.Key().Code == tea.KeyEnter && a.setupModel.step == stepConfirm {
		a.saveFromSetup()
		a.currentView = viewDashboard
		return a, nil
	}

	// Delegate to setup model
	updated, cmd := a.setupModel.Update(msg)
	a.setupModel = updated

	if a.setupModel.cancelled {
		a.currentView = viewDashboard
		return a, nil
	}
	if a.setupModel.done {
		a.saveFromSetup()
		a.currentView = viewDashboard
		return a, nil
	}

	return a, cmd
}

func (a *App) updateHelp(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return a, nil
	}

	switch {
	case keyMsg.Key().Code == tea.KeyEscape:
		a.currentView = viewDashboard
	case keyMsg.Key().Text == "?":
		a.currentView = viewDashboard
	case keyMsg.Key().Text == "q":
		a.currentView = viewDashboard
	}

	return a, nil
}

// --- Helper methods ---

// waitForResult returns a tea.Cmd that reads the next result from the channel.
func (a *App) waitForResult() tea.Cmd {
	return func() tea.Msg {
		result := <-a.resultsCh
		return checkResultMsg{result: result}
	}
}

// tickCmd returns a tea.Cmd that sends a tickMsg after one second.
func tickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// handleCheckResult processes a check result: stores it, updates the monitor
// view, recalculates status bar counts, runs the alert state machine, and
// dispatches alerts.
func (a *App) handleCheckResult(result monitor.CheckResult) {
	id := result.MonitorID

	// Store result, capping at maxResultsPerMonitor
	a.results[id] = append(a.results[id], result)
	if len(a.results[id]) > maxResultsPerMonitor {
		a.results[id] = a.results[id][len(a.results[id])-maxResultsPerMonitor:]
	}

	// Update MonitorView
	for i := range a.monitors {
		if a.monitors[i].ID == id {
			a.monitors[i].Status = result.Status
			a.monitors[i].Latency = result.Latency
			a.monitors[i].LastCheck = result.Timestamp
			a.monitors[i].Error = result.Error
			if a.monitors[i].Sparkline != nil {
				a.monitors[i].Sparkline.Push(float64(result.Latency.Milliseconds()))
			}
			break
		}
	}

	// Recalculate status bar counts
	var healthy, degraded, down int
	for _, m := range a.monitors {
		switch m.Status {
		case monitor.StatusHealthy:
			healthy++
		case monitor.StatusDegraded:
			degraded++
		case monitor.StatusDown:
			down++
		}
	}
	a.statusBar.Update(healthy, degraded, down)

	// Alert state machine
	success := result.Status == monitor.StatusHealthy || result.Status == monitor.StatusDegraded
	transition := a.alertMgr.Record(id, success)

	if transition != nil {
		// Log the transition
		monitorName := id
		for _, m := range a.monitors {
			if m.ID == id {
				monitorName = m.Name
				break
			}
		}

		msg := fmt.Sprintf("%s -> %s", transition.From, transition.To)
		isErr := transition.To == alert.StatusDown
		a.eventLog.Add(monitorName, msg, isErr)

		// Dispatch alerts based on per-monitor config
		cfg := a.configForID(id)
		if cfg != nil {
			if cfg.Alerts.Visual {
				a.visualDisp.Dispatch(*transition)
			}
			if cfg.Alerts.Desktop {
				a.desktopDisp.Dispatch(*transition)
			}
			if wh, ok := a.webhooks[id]; ok {
				wh.Dispatch(*transition)
			}
		}
	}
}

// addMonitorFromConfig creates the appropriate checker from a config, adds it
// to the engine, sets up a webhook dispatcher if configured, and appends to
// the monitors slice.
func (a *App) addMonitorFromConfig(cfg store.MonitorConfig) {
	var checker monitor.Checker

	switch cfg.Type {
	case "http":
		expectedStatus := cfg.ExpectedStatus
		if expectedStatus == 0 {
			expectedStatus = 200
		}
		checker = monitor.NewHTTPCheck(cfg.ID, cfg.Target, expectedStatus, 5*time.Second)
	case "tcp":
		checker = monitor.NewTCPCheck(cfg.ID, cfg.Target, 5*time.Second)
	case "system":
		checker = monitor.NewSystemCheck(cfg.ID)
	case "command":
		checker = monitor.NewCustomCheck(cfg.ID, cfg.Command, 5*time.Second)
	case "process":
		cpuTh := cfg.CPUThreshold
		if cpuTh == 0 {
			cpuTh = 80
		}
		memTh := cfg.MemThreshold
		if memTh == 0 {
			memTh = 80
		}
		checker = monitor.NewProcessCheck(cfg.ID, cfg.Target, cpuTh, memTh)
	default:
		a.eventLog.Add(cfg.Name, fmt.Sprintf("unknown type: %s", cfg.Type), true)
		return
	}

	interval := time.Duration(cfg.Interval) * time.Second
	if interval < time.Second {
		interval = 30 * time.Second
	}

	a.engine.Add(checker, interval)

	// Setup webhook dispatcher if configured
	if cfg.Alerts.Webhook != "" {
		a.webhooks[cfg.ID] = alert.NewWebhookDispatcher(cfg.Alerts.Webhook, 10*time.Second)
	}

	a.monitors = append(a.monitors, MonitorView{
		ID:        cfg.ID,
		Name:      cfg.Name,
		Type:      cfg.Type,
		Target:    cfg.Target,
		Sparkline: components.NewSparkline(20),
	})
}

// deleteMonitor removes a monitor from all subsystems and saves the store.
func (a *App) deleteMonitor(id string) {
	a.engine.Remove(id)
	a.alertMgr.Remove(id)
	delete(a.results, id)
	delete(a.webhooks, id)

	// Remove from monitors slice
	for i, m := range a.monitors {
		if m.ID == id {
			a.monitors = append(a.monitors[:i], a.monitors[i+1:]...)
			break
		}
	}

	// Remove from configs
	for i, c := range a.configs {
		if c.ID == id {
			a.configs = append(a.configs[:i], a.configs[i+1:]...)
			break
		}
	}

	_ = a.store.Save(a.configs)
}

// saveFromSetup extracts a config from the setup wizard and adds the monitor.
// If editing, the old monitor is deleted first.
func (a *App) saveFromSetup() {
	var id string
	if a.editing {
		id = a.detailID
		a.deleteMonitor(id)
	} else {
		id = fmt.Sprintf("mon-%d", time.Now().UnixNano())
	}

	cfg := a.setupModel.toConfig(id)

	// For command type, store the target as the command
	if cfg.Type == "command" {
		cfg.Command = cfg.Target
	}
	if cfg.Type == "process" {
		cfg.CPUThreshold = a.setupModel.cpuThresholdValue()
		cfg.MemThreshold = a.setupModel.memThresholdValue()
	}

	a.configs = append(a.configs, cfg)
	a.addMonitorFromConfig(cfg)
	_ = a.store.Save(a.configs)
}

// configForID finds a config by monitor ID, returning nil if not found.
func (a *App) configForID(id string) *store.MonitorConfig {
	for i := range a.configs {
		if a.configs[i].ID == id {
			return &a.configs[i]
		}
	}
	return nil
}

// clearPendingDelete resets the pending delete state.
func (a *App) clearPendingDelete() {
	a.pendingDelete = ""
}

// detailData returns the MonitorView, results, and sparkline for the detail view.
func (a *App) detailData() (MonitorView, []monitor.CheckResult, *components.Sparkline) {
	for _, m := range a.monitors {
		if m.ID == a.detailID {
			return m, a.results[m.ID], m.Sparkline
		}
	}
	return MonitorView{}, nil, nil
}

// renderHelp renders a static help view with polished keybinding documentation.
func renderHelp(width int) string {
	var sections []string

	header := gradientHeaderStyle.Width(width).Render("◆  Help")
	sections = append(sections, header)
	sections = append(sections, "")

	// Navigation group
	sections = append(sections, sectionHeaderStyle.Render("  NAVIGATION"))
	sections = append(sections, "")
	navBindings := []struct{ key, desc string }{
		{"j / k", "Navigate monitors"},
		{"Enter", "View monitor detail"},
		{"Esc", "Back to dashboard"},
		{"?", "Toggle help"},
	}
	for _, b := range navBindings {
		sections = append(sections, fmt.Sprintf("  %s  %s",
			helpKeyStyle.Width(16).Render(b.key),
			helpDescStyle.Render(b.desc),
		))
	}

	sections = append(sections, "")
	sections = append(sections, sectionHeaderStyle.Render("  ACTIONS"))
	sections = append(sections, "")
	actionBindings := []struct{ key, desc string }{
		{"a", "Add new monitor"},
		{"e", "Edit selected monitor"},
		{"d", "Delete selected monitor (press twice)"},
	}
	for _, b := range actionBindings {
		sections = append(sections, fmt.Sprintf("  %s  %s",
			helpKeyStyle.Width(16).Render(b.key),
			helpDescStyle.Render(b.desc),
		))
	}

	sections = append(sections, "")
	sections = append(sections, sectionHeaderStyle.Render("  VIEWS"))
	sections = append(sections, "")
	viewBindings := []struct{ key, desc string }{
		{"v", "Toggle event log"},
		{"q", "Quit"},
	}
	for _, b := range viewBindings {
		sections = append(sections, fmt.Sprintf("  %s  %s",
			helpKeyStyle.Width(16).Render(b.key),
			helpDescStyle.Render(b.desc),
		))
	}

	sections = append(sections, "")
	sections = append(sections, sectionHeaderStyle.Render("  MONITOR TYPES"))
	sections = append(sections, "")
	typeDescs := []struct{ name, desc string }{
		{"http", "Check HTTP endpoint status and response time"},
		{"tcp", "Verify TCP port is accepting connections"},
		{"system", "Monitor CPU, memory, and disk usage"},
		{"command", "Run a shell command, healthy if exit 0"},
		{"process", "Watch a named process, alert on high CPU/memory"},
	}
	for _, td := range typeDescs {
		sections = append(sections, fmt.Sprintf("  %s  %s",
			helpKeyStyle.Width(16).Render(td.name),
			helpDescStyle.Render(td.desc),
		))
	}

	sections = append(sections, "")
	help := renderHelpBar([]struct{ key, desc string }{
		{"esc", "back"},
		{"?", "close"},
		{"q", "close"},
	})
	sections = append(sections, help)

	return strings.Join(sections, "\n")
}
