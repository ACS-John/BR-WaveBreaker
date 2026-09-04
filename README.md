# WaveBreaker

A tiny standalone Windows tray utility (plain Go, no cgo, no dependencies)
that sits in the system tray, watches for the BR startup splash screen, and
presses Enter to dismiss it the instant it has focus. It is not a BR
program and is not compiled by Sad Panda / Nine Tailed Fox — it's an
ordinary Go executable, built and run independently of the ACS 5 app.

Why: launching BR (`ACS 5.exe`, or `brserver-*.exe` directly) pays a splash
screen (copyright/license/serial info, `OK`/`Wait` buttons) before anything
else happens — see `context/dev/essentials.md` §6.3 and
`.claude/skills/run-acs-5/SKILL.md`. WaveBreaker automates dismissing it
without a person needing to sit and watch for it, and without the risk of
extra keystrokes bleeding through into the app underneath (a documented
hazard of sending more than one Enter around dismissal).

## What it does

- Polls the current foreground window a few times a second.
- If that window belongs to a process named `ACS 5.exe` (or anything
  starting with `brserver`) **and** it looks like the splash screen, sends
  one real Enter keystroke (`SendInput`, not a posted message) — so it only
  ever acts on a window that is genuinely focused, matching the ask exactly.
- Sends Enter **at most once per distinct splash appearance**: once a
  window has been handled, it's never re-sent to until focus moves to a
  different window (so a slow-to-close splash doesn't get hammered with
  repeat Enters).
- Everything else — normal app screens, the Command Console, dialogs
  belonging to other programs — is left completely alone.

## How the splash is recognized (tune this if it's ever wrong)

A window is treated as the splash only when its process matches **and**
it has direct child button controls labeled `OK` and `Wait` — confirmed
present on the real splash (and, so far, nowhere else) by driving a live
launch end to end with WaveBreaker actually running against it. `-title`
(default `ACS 5`) is an extra, optional filter on top of that.

The title alone is **not** enough: BR's main frame carries the same
`"ACS 5"` title for a moment while still loading, and during that window
it is the live foreground app already receiving real keystrokes, not a
modal splash. The first version of this tool matched on title alone and,
verified live, sent three Enters instead of one — the extra two landed as
real keystrokes in the app underneath and navigated it two menus deep.
Requiring the `OK`+`Wait` buttons (a pair only the splash dialog carries)
fixed it: a second live run pressed Enter exactly once, and the app was
left sitting at its normal post-splash screen. **Never loosen this back to
a title-only match** without re-verifying live the same way.

If a BR build/config ever shows a splash with different button text,
adjust `-title` and, if needed, edit `hasOKAndWaitButtons` in `watcher.go`
— and re-verify live (see below) before trusting it again.

## Usage

```
WaveBreaker.exe [flags]
```

| Flag | Default | Meaning |
|---|---|---|
| `-procs` | `ACS 5.exe,brserver` | Comma-separated process names/prefixes to watch (case-insensitive) |
| `-title` | `ACS 5` | Extra exact-title filter the splash must also match (case-insensitive); `""` disables it |
| `-interval` | `250ms` | Poll interval |
| `-verbose` | `false` | Log each detected splash (to the console, if run from one) |

Right-click (or left-click) the tray icon for the menu:

- **Start with Windows** — a checkbox. Checking it adds a per-user
  `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` entry (value name
  `WaveBreaker`) pointing at this exe's current path, quoted; unchecking it
  removes that entry. No admin rights needed, and it only affects the
  current Windows user. The checkbox re-syncs from the registry each time
  the menu opens, so it won't drift if the entry is ever changed by hand.
- **Close** — quits. Does not auto-exit after dismissing a splash, since
  the point is to leave it running for the next launch too.

## Build

```powershell
.\build.ps1
```

Produces `WaveBreaker.exe` — a GUI-subsystem binary (no console flash),
stripped of debug symbols, using the app's `Core\Icon\acs round.ico` for both
the tray icon and the .exe's own file icon. Measured resident memory while
idle: ~12-13 MB.

The build step runs `go run github.com/akavel/rsrc@latest` to (re)generate
`rsrc_windows_amd64.syso`, embedding `acs round.ico` as Windows resource ID 1
(`go build` auto-links any `*.syso` file present in this directory) —
that one step needs network access to fetch `rsrc`; everything else builds
offline. `tray.go` loads it at runtime via `LoadIconW(hInstance, 1)`,
falling back to a stock system icon if the `.syso` wasn't linked in (e.g. a
build that skipped the rsrc step).

## Files

- `main.go` — flag parsing, wiring the watcher goroutine to the tray/message loop.
- `watcher.go` — the splash-detection and Enter-sending logic.
- `tray.go` — the hidden window, tray icon, and right-click menu.
- `startup.go` — the "Start with Windows" HKCU Run-key toggle.
- `win32.go` — the raw user32/kernel32/shell32/advapi32 bindings everything else uses.
- `rsrc_windows_amd64.syso` — generated by `build.ps1`; embeds `acs round.ico` as resource ID 1.
