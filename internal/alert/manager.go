package alert

type Status string

const (
	StatusHealthy   Status = "healthy"
	StatusDown      Status = "down"
	StatusRecovered Status = "recovered"
)

type Transition struct {
	MonitorID string
	From      Status
	To        Status
}

type monitorState struct {
	status           Status
	consecutiveFails int
	consecutiveOK    int
}

type Manager struct {
	failThreshold    int
	recoverThreshold int
	states           map[string]*monitorState
}

func NewManager(failThreshold, recoverThreshold int) *Manager {
	return &Manager{
		failThreshold:    failThreshold,
		recoverThreshold: recoverThreshold,
		states:           make(map[string]*monitorState),
	}
}

func (m *Manager) Record(monitorID string, success bool) *Transition {
	state, ok := m.states[monitorID]
	if !ok {
		state = &monitorState{status: StatusHealthy}
		m.states[monitorID] = state
	}

	if success {
		state.consecutiveFails = 0
		state.consecutiveOK++
	} else {
		state.consecutiveOK = 0
		state.consecutiveFails++
	}

	switch state.status {
	case StatusHealthy:
		if state.consecutiveFails >= m.failThreshold {
			state.status = StatusDown
			state.consecutiveFails = 0
			return &Transition{MonitorID: monitorID, From: StatusHealthy, To: StatusDown}
		}

	case StatusRecovered:
		if success {
			state.status = StatusHealthy
			state.consecutiveOK = 0
			return nil
		}
		if state.consecutiveFails >= m.failThreshold {
			state.status = StatusDown
			state.consecutiveFails = 0
			return &Transition{MonitorID: monitorID, From: StatusRecovered, To: StatusDown}
		}

	case StatusDown:
		if state.consecutiveOK >= m.recoverThreshold {
			state.status = StatusRecovered
			state.consecutiveOK = 0
			return &Transition{MonitorID: monitorID, From: StatusDown, To: StatusRecovered}
		}
	}

	return nil
}

func (m *Manager) GetStatus(monitorID string) Status {
	state, ok := m.states[monitorID]
	if !ok {
		return StatusHealthy
	}
	return state.status
}

func (m *Manager) Remove(monitorID string) {
	delete(m.states, monitorID)
}
