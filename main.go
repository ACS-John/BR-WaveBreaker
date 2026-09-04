// Command WaveBreaker sits in the system tray and waits for the BR startup
// splash screen (see README.md) to appear in the foreground, then presses
// Enter to dismiss it. It does nothing else: no BR license, no window of
// its own, negligible memory/CPU. Right-click (or left-click) the tray
// icon and choose Close to quit.
package main

import (
	"flag"
	"log"
	"runtime"
	"strings"
	"time"
)

func main() {
	procs := flag.String("procs", "ACS 5.exe,brserver", "comma-separated process names/prefixes to watch (case-insensitive)")
	title := flag.String("title", "ACS 5", "extra exact-title filter the splash must also match (case-insensitive); \"\" disables this check")
	interval := flag.Duration("interval", 250*time.Millisecond, "how often to poll the foreground window")
	verbose := flag.Bool("verbose", false, "log each detected splash to stderr")
	flag.Parse()

	cfg := Config{
		ProcNames: strings.Split(*procs, ","),
		Title:     *title,
		Interval:  *interval,
		Verbose:   *verbose,
	}

	if *verbose {
		log.Printf("WaveBreaker watching procs=%v title=%q interval=%v", cfg.ProcNames, cfg.Title, cfg.Interval)
	}

	// The window/message-loop calls below must all happen on the same OS
	// thread that created the window.
	runtime.LockOSThread()

	stop := make(chan struct{})
	go watch(cfg, stop)

	runTray("WaveBreaker - watching for BR splash screen")
	close(stop)
}
