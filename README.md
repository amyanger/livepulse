# livepulse

Real-time infrastructure monitor with a full interactive TUI dashboard. Single Go binary, no config files needed.

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/license-MIT-blue)

## Features

- **4 check types** — HTTP endpoints, TCP ports, system metrics (CPU/mem/disk), custom shell commands
- **Live TUI dashboard** — status overview, latency sparkline graphs, event log, keyboard navigation
- **Interactive setup** — add/remove/edit monitors directly from the TUI, no config files to manage
- **Alert system** — visual indicators + terminal bell, native desktop notifications, webhook POSTs (Slack, Discord, etc.)
- **Alert state machine** — configurable thresholds for down detection (3 consecutive failures) and recovery (2 consecutive successes)
- **Concurrent monitoring** — one goroutine per monitor, non-blocking checks
- **Persistent config** — monitors survive restarts via `~/.livepulse/monitors.json`

## Install

```bash
go install github.com/amyanger/livepulse/cmd/livepulse@latest
```

Or build from source:

```bash
git clone https://github.com/amyanger/livepulse.git
cd livepulse
go build -o livepulse ./cmd/livepulse
```

## Usage

```bash
./livepulse
```

Launches the TUI dashboard. Press `a` to add your first monitor.

### Keybindings

**Dashboard**

| Key | Action |
|-----|--------|
| `a` | Add new monitor |
| `e` | Edit selected monitor |
| `d` | Delete selected monitor |
| `Enter` | View monitor details |
| `j` / `k` | Navigate up/down |
| `?` | Help |
| `q` | Quit |

**Detail View**

| Key | Action |
|-----|--------|
| `Esc` | Back to dashboard |
| `e` | Edit monitor |
| `d` | Delete monitor |

**Setup Wizard**

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate options |
| `Enter` | Confirm / next step |
| `y` / `n` | Toggle yes/no options |
| `Esc` | Cancel |

## Check Types

| Type | What it does | Status logic |
|------|-------------|-------------|
| **HTTP** | GET request, validate status code + measure latency | Healthy if expected status, Down otherwise |
| **TCP** | Connect to host:port | Healthy if connection succeeds |
| **System** | CPU%, memory%, disk% via gopsutil | Degraded if any metric >90% |
| **Custom** | Run a shell command (`sh -c`) | Healthy if exit code 0 |

## Alerts

Each monitor can independently enable:

- **Visual** — terminal bell + color-coded status indicators
- **Desktop notifications** — native OS notifications on state changes
- **Webhooks** — POST JSON to a URL when a monitor goes down or recovers

Webhook payload:

```json
{
  "monitor": "Production API",
  "status": "down",
  "previous_status": "healthy",
  "timestamp": "2026-03-28T14:30:00Z"
}
```

## Architecture

```
cmd/livepulse/        Entry point
internal/
  monitor/            Checker interface, HTTP/TCP/system/custom checks, engine
  alert/              State machine, visual/desktop/webhook dispatchers
  store/              JSON persistence (~/.livepulse/monitors.json)
  tui/                Bubbletea app model, dashboard, detail, setup views
  tui/components/     Sparkline, status bar, event log
```

The monitor engine runs one goroutine per check on a configurable interval. Results flow through a buffered channel to the TUI, which updates the display and evaluates alert transitions.

## Development

```bash
# Run
go run ./cmd/livepulse

# Test
go test ./...

# Test with race detection
go test -race ./...

# Build
go build -o livepulse ./cmd/livepulse
```

## Built With

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) — Terminal styling
- [Bubbles](https://github.com/charmbracelet/bubbles) — TUI components
- [beeep](https://github.com/gen2brain/beeep) — Desktop notifications
- [gopsutil](https://github.com/shirou/gopsutil) — System metrics

## License

MIT
