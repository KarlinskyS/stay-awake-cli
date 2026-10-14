# stay

[![CI](https://github.com/KarlinskyS/stay-awake-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/KarlinskyS/stay-awake-cli/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/KarlinskyS/stay-awake-cli)](https://github.com/KarlinskyS/stay-awake-cli/releases)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Homebrew](https://img.shields.io/badge/homebrew-KarlinskyS%2Ftap%2Fstay-orange.svg)](https://github.com/KarlinskyS/homebrew-tap)

`stay` is a tiny macOS CLI utility that keeps the user session active by posting real mouse movement events through Core Graphics.

It is designed as a transparent foreground process:

```bash
stay
```

While the command is running, `stay` is active. Press `Ctrl+C` and it stops immediately.

`stay` is CLI-only. It does not install a `.app`, menu bar item, tray icon, daemon, launch agent, AppleScript, or any GUI component.

## Agent Index

```yaml
name: stay
type: cli-utility
platforms:
  - macos
architectures:
  - apple-silicon
  - intel
language: go
install:
  homebrew: brew install KarlinskyS/tap/stay
command: stay
purpose: prevent macOS user idle by posting real mouse movement events
input-method: Core Graphics mouse movement events
requires:
  - macOS Accessibility permission
does-not-use:
  - GUI app
  - menu bar
  - tray icon
  - daemon
  - launch agent
  - AppleScript
  - external GUI automation
runtime-model: long-running foreground terminal process until Ctrl+C, SIGINT, SIGTERM, or optional --wait-agent idle stop
default-interval: 60s
default-distance: 1 display point
primary-use-cases:
  - macOS idle prevention
  - transparent CLI mouse mover
  - foreground terminal alternative to GUI mouse mover apps
  - keep Mac awake until Claude, Grok, or Codex finishes an overnight task
```

Search keywords: `macOS idle prevention`, `mouse mover CLI`, `keep Mac awake by mouse movement`, `Core Graphics mouse event`, `Accessibility permission`, `Apple Silicon CLI`, `no GUI mouse mover`, `foreground terminal process`.

## Discovery Notes

If you are looking for a CLI alternative to a mouse jiggler, mouse mover, or GUI keep-awake app on macOS, `stay` is built for that use case. It keeps the behavior visible in a terminal, uses native Core Graphics mouse events, and avoids AppleScript or background agents.

Related search terms: macOS mouse jiggler CLI, mouse mover for Mac terminal, keep Mac session active, prevent macOS idle with mouse movement, no-GUI mouse mover, Apple Silicon mouse mover.

## Why

Some GUI mouse mover apps are hard to inspect, hard to debug, and awkward to run in controlled environments. `stay` keeps the behavior explicit:

- one terminal command starts it;
- terminal logs show every decision;
- `Ctrl+C`, `SIGINT`, or `SIGTERM` stops it;
- the cursor returns to its original position after every synthetic movement;
- there is no accumulated cursor drift.

The implementation is inspired by the general idea of periodic mouse movement tools, including `automatic-mouse-mover`, but no source code from that project is copied here.

## Requirements

- macOS.
- Apple Silicon or Intel Mac.
- Go 1.22+ if building from source.
- Accessibility permission for either:
  - the terminal application that runs `stay`; or
  - the installed `stay` binary.

`stay` uses macOS ApplicationServices/Core Graphics APIs. It does not use AppleScript.

## Install

Recommended install method:

```bash
brew install KarlinskyS/tap/stay
```

This installs the `stay` command from the public Homebrew tap:

```text
KarlinskyS/homebrew-tap
```

Then run:

```bash
stay
```

You do not need to install Go manually for the Homebrew flow. Homebrew builds `stay` from source and installs any build dependencies it needs.

Upgrade later:

```bash
brew upgrade KarlinskyS/tap/stay
```

Uninstall:

```bash
brew uninstall stay
```

## Build From Source

Build and install into `$HOME/.local/bin`:

```bash
make build
make install
```

Make sure `$HOME/.local/bin` is in your `PATH`:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

Install into `/usr/local/bin` instead:

```bash
sudo make install PREFIX=/usr/local
```

## Run

```bash
stay
```

Default behavior:

- checks activity every `60s`;
- moves the cursor by `1` display point;
- skips synthetic movement if real mouse or keyboard activity happened during the interval;
- moves to a nearby valid on-screen point and then returns to the original cursor position.

Example output:

```text
13:42:00 Stay started
Interval: 60s
Move distance: 1 display point(s)
Press Ctrl+C to stop

13:43:00 idle for 60s - cursor moved
13:44:00 user activity detected - skipped
13:45:00 idle for 60s - cursor moved
13:45:12 stopped
```

## Options

```bash
stay --interval 60s
stay --distance 1
stay --always
stay --verbose
stay --version
stay --help
stay --wait-agent
stay --wait-claude
stay --wait-codex
stay --wait-grok
```

| Option | Description |
| --- | --- |
| `--interval` | Check interval. Accepts Go duration values such as `60s`, `1m`, `500ms`. A bare integer is treated as seconds. |
| `--distance` | Cursor movement distance in macOS display points. Default: `1`. |
| `--always` | Move every interval, even if user activity was detected. |
| `--verbose` | Print more detailed diagnostics. |
| `--version` | Print version and exit. |
| `--help` | Print usage and exit. |
| `--wait-agent` | Keep the session awake until Claude, Grok, and Codex finish their current work, then exit. |
| `--wait-claude` | Same, watching only Claude Code. |
| `--wait-grok` | Same, watching only Grok. |
| `--wait-codex` | Same, watching only Codex. |
| `--wait-start` | How long to wait for a working agent to appear. Default: `10m`. |
| `--wait-idle` | How long agents must stay idle before `stay` exits. Default: `60s`. |

Examples:

```bash
stay --interval 30s
stay --interval 5s --verbose
stay --distance 2
stay --always
stay --wait-claude
stay --wait-agent --wait-idle 30s
```

## Wait for an agent

`stay` can keep the Mac awake for an overnight Claude, Grok, or Codex task, then exit so the machine can sleep.

Start the task, then in another terminal:

```bash
stay --wait-claude
```

Watch every supported agent:

```bash
stay --wait-agent
```

Combine specific agents:

```bash
stay --wait-claude --wait-grok
```

`stay` watches whether the agent is still working, not whether the process is still running. Interactive CLIs sit at a prompt after a turn finishes; that idle prompt is the stop signal. Codex background `app-server` daemons are ignored.

A Claude session is working while its `~/.claude/sessions/<pid>.json` status is anything other than `idle` (`busy`, `shell`, `waiting`, ...). Grok is working until `events.jsonl` records `turn_ended`. Codex is working while a recent rollout file's last turn boundary is `task_started`.

If no working agent is visible yet, `stay` waits up to `--wait-start` (default 10 minutes) for one to appear. After work is seen and then goes idle, `stay` waits `--wait-idle` (default 60 seconds) so a brief pause between tools does not look like a finished task. Then it logs `agents finished - stopping` and exits.

A permission prompt counts as still working. If the overnight task stops to ask for approval, `stay` keeps the session awake until you answer or stop it with Ctrl+C.

## Accessibility Permission

On first run, macOS may block synthetic input events until Accessibility permission is granted.

If permission is missing, `stay` exits with:

```text
Stay requires Accessibility permission.
Open System Settings -> Privacy & Security -> Accessibility
and allow your terminal application or the stay binary.
```

To grant permission:

1. Open `System Settings`.
2. Go to `Privacy & Security`.
3. Open `Accessibility`.
4. Enable your terminal application, for example `Terminal`, `iTerm2`, `Ghostty`, `WezTerm`, or `Alacritty`.
5. Run `stay` again.

If you run the installed binary directly and macOS shows the binary as the requesting process, allow `stay` itself.

## How It Works

Every interval, `stay` asks macOS how long it has been since the last input event.

If the user has been idle for at least the configured interval, `stay`:

1. reads the current cursor position;
2. reads the active display bounds;
3. chooses a nearby valid point on any active display;
4. posts a Core Graphics `kCGEventMouseMoved` event;
5. waits briefly;
6. posts another mouse move event back to the original position.

This handles screen edges and multi-monitor layouts, including displays with negative coordinates.

Because synthetic movement itself resets the macOS idle timer, `stay` tracks its own last synthetic movement and distinguishes it from real user activity.

## Diagnostics

Run with a short interval while testing:

```bash
stay --interval 5s --verbose
```

Check macOS power assertions:

```bash
pmset -g assertions
```

After synthetic movement, macOS should report recent user activity, commonly through a `UserIsActive` assertion owned by `WindowServer`.

Useful checks:

```bash
stay --version
stay --help
make test
make build
```

## Troubleshooting

### `command not found: stay`

The install path is probably not in `PATH`.

Run directly:

```bash
$HOME/.local/bin/stay
```

Or add it to your shell profile:

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Accessibility permission was granted, but movement still fails

Try the following:

- restart the terminal application;
- remove and re-add the terminal application in Accessibility settings;
- if launching `/usr/local/bin/stay` or `$HOME/.local/bin/stay` directly, allow the binary if macOS lists it separately;
- run `stay --interval 5s --verbose` and inspect the logs.

### Cursor moves but a remote desktop session does not see it

Remote desktop clients decide which local input events are forwarded into the remote session. `stay` can only generate local macOS mouse movement. The remote client must accept and forward that movement.

## Uninstall

If installed into `$HOME/.local/bin`:

```bash
rm -f "$HOME/.local/bin/stay"
```

If installed into `/usr/local/bin`:

```bash
sudo rm -f /usr/local/bin/stay
```

You can also remove the Accessibility permission from `System Settings -> Privacy & Security -> Accessibility`.

## Development

```bash
make test
make build
bin/stay --help
bin/stay --interval 5s --verbose
```

Project layout:

```text
.
├── cmd/stay/main.go
├── internal/activity
├── internal/agent
├── internal/cli
├── internal/mover
├── go.mod
├── Makefile
├── README.md
└── LICENSE
```

Manual smoke checklist:

- `stay` starts a long-running foreground process.
- Terminal output clearly shows that the process is active.
- `stay --wait-claude` exits after Claude goes idle.
- `Ctrl+C` stops the process and logs shutdown.
- `SIGTERM` stops the process and logs shutdown.
- Each actual synthetic movement is logged.
- Cursor returns to the original position after movement.
- Repeated movement does not accumulate cursor drift.
- Movement works on the primary monitor.
- Movement works on an additional monitor.
- Cursor near screen edges still moves to a valid on-screen point and returns.
- Missing Accessibility permission shows the documented error.
- `pmset -g assertions` or another system indicator sees user activity after synthetic movement.

## Policy

Use `stay` only in ways that comply with your organization policies, contracts, and local rules. This tool is intentionally transparent so its behavior is easy to inspect and reason about.

## License

MIT.
