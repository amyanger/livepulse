# livepulse

Real-time infrastructure monitor with a full interactive TUI dashboard. Single Go binary, no config files.

## Commands

```bash
# Build
go build -o livepulse ./cmd/livepulse

# Run
go run ./cmd/livepulse

# Test
go test ./...

# Test with race detection
go test -race ./...

# Lint (if golangci-lint installed)
golangci-lint run
```

## Architecture

```
cmd/livepulse/main.go    -> entry point
internal/monitor/         -> check engine + check type implementations
internal/tui/             -> bubbletea TUI (dashboard, detail, setup views)
internal/tui/components/  -> reusable TUI components (sparkline, statusbar, eventlog)
internal/alert/           -> alert state machine + dispatchers (visual, desktop, webhook)
internal/store/           -> persistence layer (~/.livepulse/monitors.json)
```

## Key Libraries

- `bubbletea` — TUI framework (Elm architecture)
- `lipgloss` — TUI styling/layout
- `bubbles` — TUI components (tables, spinners, text inputs)
- `beeep` — cross-platform desktop notifications
- `gopsutil` — system metrics (CPU/mem/disk)

## Code Style

- Standard Go project layout: `cmd/` for entrypoints, `internal/` for private packages
- All check types implement the `Checker` interface in `internal/monitor/check.go`
- Use `context.Context` for goroutine lifecycle management
- Channels for communication between monitor goroutines and state store
- Error handling: wrap errors with `fmt.Errorf("context: %w", err)`
- No global state — pass dependencies explicitly

## Concurrency

- One goroutine per monitor with `time.Ticker`
- Buffered result channel to central state store
- TUI subscribes via bubbletea commands
- Clean shutdown via `context.Context` cancellation

## Testing

- Table-driven tests for check implementations
- Mock `Checker` interface for engine tests
- Use `httptest.NewServer` for HTTP check tests
- Test alert state transitions explicitly

## Gotchas

- `bubbletea` programs must call `p.Run()` on the main goroutine
- `lipgloss` width calculations can be off with wide Unicode — use ASCII for status indicators
- `gopsutil` requires cgo on some platforms — test cross-platform builds
- Webhook POSTs must be non-blocking (use goroutine with timeout) to avoid stalling the monitor loop
- Desktop notifications may require permissions on macOS (System Settings -> Notifications)

## Monitor State Persistence

- Monitors saved to `~/.livepulse/monitors.json` on every change
- Check results are ephemeral (not persisted across restarts)
- Alert thresholds: 3 consecutive failures = down, 2 consecutive successes = recovered
