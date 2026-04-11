package tui

import (
	"fmt"
	"strconv"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/bubbles/v2/textinput"

	"github.com/arjunmyanger/livepulse/internal/store"
)

// setupStep represents the current step in the setup wizard.
type setupStep int

const (
	stepType         setupStep = iota
	stepTarget
	stepName
	stepInterval
	stepCPUThreshold
	stepMemThreshold
	stepAlertVisual
	stepAlertDesktop
	stepWebhook
	stepConfirm
)

// monitorTypes lists the available monitor types for selection.
var monitorTypes = []string{"http", "tcp", "system", "command", "process"}

// stepLabels maps each step to a display label for the progress indicator.
var stepLabels = []string{
	"Type", "Target", "Name", "Interval",
	"CPU Threshold", "Mem Threshold",
	"Visual Alert", "Desktop Alert", "Webhook", "Confirm",
}

// setupModel is the Bubble Tea model for the multi-step setup wizard.
type setupModel struct {
	step         setupStep
	typeIdx      int
	targetInput  textinput.Model
	nameInput    textinput.Model
	intervalInput textinput.Model
	webhookInput textinput.Model
	alertVisual       bool
	alertDesktop      bool
	cpuThresholdInput textinput.Model
	memThresholdInput textinput.Model
	done              bool
	cancelled         bool
}

// newSetupModel creates a new setup wizard. If existing is non-nil, the fields
// are pre-filled for editing.
func newSetupModel(existing *store.MonitorConfig) setupModel {
	target := textinput.New()
	target.Placeholder = "e.g. https://example.com or host:port"
	target.CharLimit = 256

	name := textinput.New()
	name.Placeholder = "e.g. My API Server"
	name.CharLimit = 64

	interval := textinput.New()
	interval.Placeholder = "seconds (default: 30)"
	interval.CharLimit = 6

	webhook := textinput.New()
	webhook.Placeholder = "https://hooks.example.com/alert (optional)"
	webhook.CharLimit = 512

	cpuTh := textinput.New()
	cpuTh.Placeholder = "% (default: 80)"
	cpuTh.CharLimit = 3

	memTh := textinput.New()
	memTh.Placeholder = "% (default: 80)"
	memTh.CharLimit = 3

	m := setupModel{
		step:              stepType,
		targetInput:       target,
		nameInput:         name,
		intervalInput:     interval,
		webhookInput:      webhook,
		cpuThresholdInput: cpuTh,
		memThresholdInput: memTh,
	}

	// Pre-fill for editing
	if existing != nil {
		for i, t := range monitorTypes {
			if t == existing.Type {
				m.typeIdx = i
				break
			}
		}
		m.targetInput.SetValue(existing.Target)
		m.nameInput.SetValue(existing.Name)
		if existing.Interval > 0 {
			m.intervalInput.SetValue(strconv.Itoa(existing.Interval))
		}
		m.alertVisual = existing.Alerts.Visual
		m.alertDesktop = existing.Alerts.Desktop
		m.webhookInput.SetValue(existing.Alerts.Webhook)
		if existing.CPUThreshold > 0 {
			m.cpuThresholdInput.SetValue(strconv.FormatFloat(existing.CPUThreshold, 'f', 0, 64))
		}
		if existing.MemThreshold > 0 {
			m.memThresholdInput.SetValue(strconv.FormatFloat(existing.MemThreshold, 'f', 0, 64))
		}
	}

	return m
}

// Update handles input for the current wizard step.
func (m setupModel) Update(msg tea.Msg) (setupModel, tea.Cmd) {
	keyMsg, isKey := msg.(tea.KeyPressMsg)
	if !isKey {
		return m.updateTextInputs(msg)
	}

	key := keyMsg.Key()

	// Global: escape cancels
	if key.Code == tea.KeyEscape {
		m.cancelled = true
		return m, nil
	}

	switch m.step {
	case stepType:
		return m.updateTypeStep(key)
	case stepTarget:
		return m.updateTextStep(&m.targetInput, key, msg)
	case stepName:
		return m.updateTextStep(&m.nameInput, key, msg)
	case stepInterval:
		return m.updateTextStep(&m.intervalInput, key, msg)
	case stepCPUThreshold:
		return m.updateTextStep(&m.cpuThresholdInput, key, msg)
	case stepMemThreshold:
		return m.updateTextStep(&m.memThresholdInput, key, msg)
	case stepAlertVisual:
		return m.updateBoolStep(&m.alertVisual, key)
	case stepAlertDesktop:
		return m.updateBoolStep(&m.alertDesktop, key)
	case stepWebhook:
		return m.updateTextStep(&m.webhookInput, key, msg)
	case stepConfirm:
		return m.updateConfirmStep(key)
	}

	return m, nil
}

// updateTypeStep handles j/k/up/down navigation and enter selection.
func (m setupModel) updateTypeStep(key tea.Key) (setupModel, tea.Cmd) {
	switch key.Code {
	case tea.KeyUp:
		if m.typeIdx > 0 {
			m.typeIdx--
		}
	case tea.KeyDown:
		if m.typeIdx < len(monitorTypes)-1 {
			m.typeIdx++
		}
	case tea.KeyEnter:
		m.step = stepTarget
		return m, m.targetInput.Focus()
	default:
		switch key.Text {
		case "k":
			if m.typeIdx > 0 {
				m.typeIdx--
			}
		case "j":
			if m.typeIdx < len(monitorTypes)-1 {
				m.typeIdx++
			}
		}
	}
	return m, nil
}

// updateTextStep forwards input to a text input field; enter advances to next step.
func (m setupModel) updateTextStep(input *textinput.Model, key tea.Key, msg tea.Msg) (setupModel, tea.Cmd) {
	if key.Code == tea.KeyEnter {
		input.Blur()
		m.step = m.nextStep(m.step)
		return m, m.focusCurrentInput()
	}

	updated, cmd := input.Update(msg)
	*input = updated
	return m, cmd
}

// updateBoolStep handles y/n keys for boolean fields.
func (m setupModel) updateBoolStep(field *bool, key tea.Key) (setupModel, tea.Cmd) {
	switch key.Text {
	case "y":
		*field = true
		m.step = m.nextStep(m.step)
		return m, m.focusCurrentInput()
	case "n":
		*field = false
		m.step = m.nextStep(m.step)
		return m, m.focusCurrentInput()
	case "Y":
		*field = true
		m.step = m.nextStep(m.step)
		return m, m.focusCurrentInput()
	case "N":
		*field = false
		m.step = m.nextStep(m.step)
		return m, m.focusCurrentInput()
	case "":
		// Enter confirms the current value (default false)
		if key.Code == tea.KeyEnter {
			m.step = m.nextStep(m.step)
			return m, m.focusCurrentInput()
		}
	}
	return m, nil
}

// updateConfirmStep handles enter to confirm or backspace to go back.
func (m setupModel) updateConfirmStep(key tea.Key) (setupModel, tea.Cmd) {
	switch key.Code {
	case tea.KeyEnter:
		m.done = true
	case tea.KeyBackspace:
		m.step = m.prevStep(m.step)
		return m, m.focusCurrentInput()
	}
	return m, nil
}

// updateTextInputs forwards non-key messages to the currently focused text input.
func (m setupModel) updateTextInputs(msg tea.Msg) (setupModel, tea.Cmd) {
	var cmd tea.Cmd
	switch m.step {
	case stepTarget:
		m.targetInput, cmd = m.targetInput.Update(msg)
	case stepName:
		m.nameInput, cmd = m.nameInput.Update(msg)
	case stepInterval:
		m.intervalInput, cmd = m.intervalInput.Update(msg)
	case stepCPUThreshold:
		m.cpuThresholdInput, cmd = m.cpuThresholdInput.Update(msg)
	case stepMemThreshold:
		m.memThresholdInput, cmd = m.memThresholdInput.Update(msg)
	case stepWebhook:
		m.webhookInput, cmd = m.webhookInput.Update(msg)
	}
	return m, cmd
}

// focusCurrentInput focuses the text input for the current step, if applicable.
func (m setupModel) focusCurrentInput() tea.Cmd {
	switch m.step {
	case stepTarget:
		return m.targetInput.Focus()
	case stepName:
		return m.nameInput.Focus()
	case stepInterval:
		return m.intervalInput.Focus()
	case stepCPUThreshold:
		return m.cpuThresholdInput.Focus()
	case stepMemThreshold:
		return m.memThresholdInput.Focus()
	case stepWebhook:
		return m.webhookInput.Focus()
	}
	return nil
}

// nextStep returns the step after the given one, skipping threshold steps for non-process types.
func (m setupModel) nextStep(current setupStep) setupStep {
	next := current + 1
	if monitorTypes[m.typeIdx] != "process" {
		if next == stepCPUThreshold || next == stepMemThreshold {
			return stepAlertVisual
		}
	}
	return next
}

// prevStep returns the step before the given one, skipping threshold steps for non-process types.
func (m setupModel) prevStep(current setupStep) setupStep {
	prev := current - 1
	if monitorTypes[m.typeIdx] != "process" {
		if prev == stepMemThreshold || prev == stepCPUThreshold {
			return stepInterval
		}
	}
	return prev
}

// View renders the current wizard step with progress indicators.
func (m setupModel) View() string {
	var sections []string

	header := headerStyle.Render("Setup Monitor")
	sections = append(sections, header)
	sections = append(sections, "")

	// Progress bar
	sections = append(sections, m.renderProgress())
	sections = append(sections, "")

	// Current step content
	switch m.step {
	case stepType:
		sections = append(sections, titleStyle.Render("Select Monitor Type"))
		sections = append(sections, "")
		for i, t := range monitorTypes {
			cursor := "  "
			if i == m.typeIdx {
				cursor = "> "
				sections = append(sections, selectedStyle.Render(cursor+t))
			} else {
				sections = append(sections, helpStyle.Render(cursor+t))
			}
		}
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  j/k:navigate  enter:select  esc:cancel"))

	case stepTarget:
		title := "Enter Target"
		if monitorTypes[m.typeIdx] == "process" {
			title = "Enter Process Name"
		}
		sections = append(sections, titleStyle.Render(title))
		sections = append(sections, "")
		sections = append(sections, "  "+m.targetInput.View())
		sections = append(sections, "")
		hint := helpStyle.Render("  enter:next  esc:cancel")
		if monitorTypes[m.typeIdx] == "process" {
			hint = helpStyle.Render("  e.g. nginx, postgres, redis  |  enter:next  esc:cancel")
		}
		sections = append(sections, hint)

	case stepName:
		sections = append(sections, titleStyle.Render("Enter Name"))
		sections = append(sections, "")
		sections = append(sections, "  "+m.nameInput.View())
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  enter:next  esc:cancel"))

	case stepInterval:
		sections = append(sections, titleStyle.Render("Check Interval (seconds)"))
		sections = append(sections, "")
		sections = append(sections, "  "+m.intervalInput.View())
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  enter:next  esc:cancel"))

	case stepCPUThreshold:
		sections = append(sections, titleStyle.Render("CPU Threshold (%)"))
		sections = append(sections, "")
		sections = append(sections, "  "+m.cpuThresholdInput.View())
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  enter:next  esc:cancel"))

	case stepMemThreshold:
		sections = append(sections, titleStyle.Render("Memory Threshold (%)"))
		sections = append(sections, "")
		sections = append(sections, "  "+m.memThresholdInput.View())
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  enter:next  esc:cancel"))

	case stepAlertVisual:
		sections = append(sections, titleStyle.Render("Enable Visual Alerts?"))
		sections = append(sections, "")
		sections = append(sections, fmt.Sprintf("  Current: %s", boolLabel(m.alertVisual)))
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  y:yes  n:no  esc:cancel"))

	case stepAlertDesktop:
		sections = append(sections, titleStyle.Render("Enable Desktop Notifications?"))
		sections = append(sections, "")
		sections = append(sections, fmt.Sprintf("  Current: %s", boolLabel(m.alertDesktop)))
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  y:yes  n:no  esc:cancel"))

	case stepWebhook:
		sections = append(sections, titleStyle.Render("Webhook URL (optional)"))
		sections = append(sections, "")
		sections = append(sections, "  "+m.webhookInput.View())
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  enter:next  esc:cancel"))

	case stepConfirm:
		sections = append(sections, titleStyle.Render("Confirm Monitor"))
		sections = append(sections, "")
		sections = append(sections, m.renderSummary())
		sections = append(sections, "")
		sections = append(sections, helpStyle.Render("  enter:save  backspace:back  esc:cancel"))
	}

	return strings.Join(sections, "\n")
}

// renderProgress renders the step progress indicator line.
func (m setupModel) renderProgress() string {
	var parts []string
	for i, label := range stepLabels {
		step := setupStep(i)
		// Skip threshold steps for non-process types
		if monitorTypes[m.typeIdx] != "process" && (step == stepCPUThreshold || step == stepMemThreshold) {
			continue
		}
		switch {
		case step == m.step:
			parts = append(parts, titleStyle.Render("-> "+label))
		case step < m.step:
			parts = append(parts, healthyStyle.Render("v "+label))
		default:
			parts = append(parts, helpStyle.Render("  "+label))
		}
	}
	return "  " + strings.Join(parts, "  ")
}

// renderSummary shows all configured values for confirmation.
func (m setupModel) renderSummary() string {
	interval := m.intervalValue()
	lines := []string{
		fmt.Sprintf("  %s %s", labelStyle.Render("Type"), valueStyle.Render(monitorTypes[m.typeIdx])),
		fmt.Sprintf("  %s %s", labelStyle.Render("Target"), valueStyle.Render(m.targetInput.Value())),
		fmt.Sprintf("  %s %s", labelStyle.Render("Name"), valueStyle.Render(m.nameInput.Value())),
		fmt.Sprintf("  %s %s", labelStyle.Render("Interval"), valueStyle.Render(fmt.Sprintf("%ds", interval))),
	}
	if monitorTypes[m.typeIdx] == "process" {
		lines = append(lines,
			fmt.Sprintf("  %s %s", labelStyle.Render("CPU Threshold"), valueStyle.Render(fmt.Sprintf("%.0f%%", m.cpuThresholdValue()))),
			fmt.Sprintf("  %s %s", labelStyle.Render("Mem Threshold"), valueStyle.Render(fmt.Sprintf("%.0f%%", m.memThresholdValue()))),
		)
	}
	lines = append(lines,
		fmt.Sprintf("  %s %s", labelStyle.Render("Visual Alert"), valueStyle.Render(boolLabel(m.alertVisual))),
		fmt.Sprintf("  %s %s", labelStyle.Render("Desktop Alert"), valueStyle.Render(boolLabel(m.alertDesktop))),
		fmt.Sprintf("  %s %s", labelStyle.Render("Webhook"), valueStyle.Render(m.webhookInput.Value())),
	)
	return strings.Join(lines, "\n")
}

// cpuThresholdValue parses the CPU threshold input, defaulting to 80%.
func (m setupModel) cpuThresholdValue() float64 {
	if v, err := strconv.ParseFloat(m.cpuThresholdInput.Value(), 64); err == nil && v > 0 {
		return v
	}
	return 80
}

// memThresholdValue parses the memory threshold input, defaulting to 80%.
func (m setupModel) memThresholdValue() float64 {
	if v, err := strconv.ParseFloat(m.memThresholdInput.Value(), 64); err == nil && v > 0 {
		return v
	}
	return 80
}

// toConfig converts the wizard state into a MonitorConfig.
func (m setupModel) toConfig(id string) store.MonitorConfig {
	return store.MonitorConfig{
		ID:       id,
		Name:     m.nameInput.Value(),
		Type:     monitorTypes[m.typeIdx],
		Target:   m.targetInput.Value(),
		Interval: m.intervalValue(),
		Alerts: store.AlertConfig{
			Visual:  m.alertVisual,
			Desktop: m.alertDesktop,
			Webhook: m.webhookInput.Value(),
		},
	}
}

// intervalValue parses the interval input, defaulting to 30 seconds.
func (m setupModel) intervalValue() int {
	if v, err := strconv.Atoi(m.intervalInput.Value()); err == nil && v > 0 {
		return v
	}
	return 30
}

// boolLabel returns "yes" or "no" for display.
func boolLabel(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
