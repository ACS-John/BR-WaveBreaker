package main

import (
	"log"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

// Config controls how a BR splash screen is recognized. All of it is
// tunable from the command line (see main.go) rather than hardcoded,
// since the splash window's exact title/buttons were only verified once
// (see README.md) and may vary by BR build or config.
type Config struct {
	ProcNames []string      // process image base names / prefixes to match, case-insensitive
	Title     string        // optional extra exact-title filter, case-insensitive; "" disables it
	Interval  time.Duration // poll interval
	Verbose   bool
}

// watch polls the foreground window and presses Enter, exactly once per
// distinct splash appearance, when it looks like the BR splash and
// currently has focus. It never returns; run it in its own goroutine.
func watch(cfg Config, stop <-chan struct{}) {
	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	var lastHandled syscall.Handle

	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
		}

		hwnd := getForegroundWindow()
		if hwnd == 0 {
			continue
		}

		if hwnd != lastHandled {
			// Focus moved to a different window than the one we last
			// pressed Enter on; re-arm so a future splash is handled too.
			lastHandled = 0
		}
		if hwnd == lastHandled {
			continue
		}

		if !processMatches(hwnd, cfg.ProcNames) {
			continue
		}

		// The OK+Wait button pair is the load-bearing signal: BR's main
		// frame carries the same "ACS 5" title as the splash for a moment
		// while it is still loading and *is* the live foreground window
		// receiving real keystrokes at that point -- confirmed the hard
		// way, sending Enter on title alone bled three keystrokes into a
		// running session (see README.md). Only the actual splash dialog
		// has these buttons, so require it unconditionally.
		if !hasOKAndWaitButtons(hwnd) {
			continue
		}

		title := getWindowText(hwnd)
		if cfg.Title != "" && !strings.EqualFold(strings.TrimSpace(title), cfg.Title) {
			continue
		}

		// Re-check foreground right before sending -- the whole point is to
		// only ever send Enter to a window that is actually in focus.
		if getForegroundWindow() != hwnd {
			continue
		}

		if cfg.Verbose {
			log.Printf("splash detected: hwnd=%v title=%q", hwnd, title)
		}

		sendEnterKey()
		lastHandled = hwnd
	}
}

func processMatches(hwnd syscall.Handle, names []string) bool {
	pid := getWindowProcessId(hwnd)
	if pid == 0 {
		return false
	}
	h := openProcess(processQueryLimitedInformation, pid)
	if h == 0 {
		return false
	}
	defer closeHandle(h)

	path := queryFullProcessImageName(h)
	if path == "" {
		return false
	}
	base := strings.ToLower(filepath.Base(path))

	for _, n := range names {
		n = strings.ToLower(strings.TrimSpace(n))
		if n == "" {
			continue
		}
		if strings.HasPrefix(base, n) || base == n {
			return true
		}
	}
	return false
}

// hasOKAndWaitButtons reports whether hwnd has direct child button controls
// labeled "OK" and "Wait" -- the pair confirmed present on the BR splash
// screen and nowhere else (see README.md). This is the required signal;
// title alone is not specific enough (see the caller).
func hasOKAndWaitButtons(hwnd syscall.Handle) bool {
	var foundOK, foundWait bool
	cb := syscall.NewCallback(func(child syscall.Handle, _ uintptr) uintptr {
		if !strings.EqualFold(getClassName(child), "Button") {
			return 1 // continue enumeration
		}
		switch strings.ToLower(strings.TrimSpace(getWindowText(child))) {
		case "ok":
			foundOK = true
		case "wait":
			foundWait = true
		}
		return 1
	})
	enumChildWindows(hwnd, cb, 0)
	return foundOK && foundWait
}
