# livepulse — Real-Time Infrastructure Monitor

## Overview

A single Go binary that provides a full interactive TUI dashboard for real-time infrastructure monitoring. No config files needed — monitors are added/removed/edited directly from the TUI.

## Architecture

### Core Components

1. **Monitor Engine** — concurrent health checks via goroutines, one per monitor on configurable intervals
2. **Check Types**:
   - HTTP endpoints (status codes + latency)
   - TCP ports (open/responding check for databases, Redis, etc.)
   - System metrics (CPU/memory/disk of local machine)
   - Custom checks (user-defined shell commands, exit code evaluation)
3. **TUI Dashboard** — built with `bubbletea` + `lipgloss` + `bubbles`
   - Live-updating status overview panel
   - Latency sparkline graphs
   - Event log
   - Detail views per monitor
   - Keyboard navigation
4. **Interactive Setup** — add/remove/edit monitors from within the TUI
   - State persists to `~/.livepulse/monitors.json`
   - Monitors survive restarts
5. **Alert System** — three tiers, configurable per monitor:
   - Visual: color shifts, flashing indicators, terminal bell
   - Desktop notifications: native OS notifications via `beeep`
   - Webhooks: POST to Slack/Discord/custom URLs on state changes

### Data Flow

```
Monitor Engine (goroutines per check)
  -> Results Channel
    -> State Store (in-memory, persisted to disk)
      -> TUI renders from state
      -> Alert Manager evaluates state transitions -> fires alerts
```

### Key Libraries

| Library | Purpose |
|---------|---------|
| `bubbletea` | TUI framework (Elm-architecture) |
| `lipgloss` | TUI styling and layout |
| `bubbles` | TUI components (tables, spinners, text inputs) |
| `beeep` | Cross-platform desktop notifications |
| `gopsutil` | System metrics (CPU/mem/disk) |

### Project Structure

```
livepulse/
├── cmd/
│   └── livepulse/
│       └── main.go              # Entry point
├── internal/
│   ├── monitor/
│   │   ├── engine.go            # Monitor engine, scheduling
│   │   ├── check.go             # Check interface + types
│   │   ├── http.go              # HTTP check implementation
│   │   ├── tcp.go               # TCP check implementation
│   │   ├── system.go            # System metrics check
│   │   └── custom.go            # Custom shell command check
│   ├── tui/
│   │   ├── app.go               # Main bubbletea model
│   │   ├── dashboard.go         # Dashboard view
│   │   ├── detail.go            # Monitor detail view
│   │   ├── setup.go             # Interactive add/edit forms
│   │   ├── styles.go            # lipgloss styles
│   │   └── components/
│   │       ├── sparkline.go     # Latency sparkline graph
│   │       ├── statusbar.go     # Status bar component
│   │       └── eventlog.go      # Event log component
│   ├── alert/
│   │   ├── manager.go           # Alert state machine
│   │   ├── visual.go            # Terminal bell, color changes
│   │   ├── desktop.go           # Desktop notifications
│   │   └── webhook.go           # Webhook dispatcher
│   └── store/
│       └── store.go             # Persist/load monitors.json
├── docs/
│   └── plans/
├── go.mod
├── go.sum
```

### State Management

- In-memory state store holds all monitor configs + recent check results (last 100 per monitor)
- State persisted to `~/.livepulse/monitors.json` on every add/remove/edit
- Check results are ephemeral (not persisted across restarts)
- Alert manager tracks state transitions: `healthy -> degraded -> down -> recovered`

### Alert State Machine

```
healthy -- failure threshold met --> down
down -- success threshold met --> recovered
recovered -- (auto after display) --> healthy
any -- consecutive slow responses --> degraded
```

Default thresholds: 3 consecutive failures = down, 2 consecutive successes = recovered.

### Concurrency Model

- Each monitor runs in its own goroutine with a `time.Ticker`
- Results sent via buffered channel to central state store
- TUI subscribes to state updates via `bubbletea` commands
- `context.Context` used for clean shutdown of all goroutines

### Interactive Setup Flow

From the TUI, user can:
1. Press `a` to add a new monitor -> form wizard (type -> URL/host -> interval -> alert config)
2. Press `e` on selected monitor to edit
3. Press `d` to delete (with confirmation)
4. Press `?` for help overlay

### Webhook Payload

```json
{
  "monitor": "Production API",
  "type": "http",
  "target": "https://api.example.com/health",
  "status": "down",
  "previous_status": "healthy",
  "latency_ms": 5023,
  "error": "timeout after 5s",
  "timestamp": "2026-03-28T14:30:00Z"
}
```
