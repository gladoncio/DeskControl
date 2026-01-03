package main

import (
	"flag"
	"log"
	"time"

	"deskcontrol/daemon/internal/loghub"
)

func main() {
	tray := flag.Bool("tray", false, "start hidden in system tray")
	flag.Parse()

	// timestamps siempre
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)

	opts := UIOpts{
		StartInTray: *tray,
		AppRunName:  "DeskControl",
		MaxUILines:  100,
		Tick:        250 * time.Millisecond,
	}

	// Hub para la pestaña Logs (y también para log early)
	hub := loghub.New(opts.MaxUILines)

	// Logging ANTES de UI (./logs al lado del exe)
	closeFn, err := InitEarlyLogging(hub)
	if err != nil {
		// si esto falla, al menos no reventamos la app
		// (pero idealmente debes correr el exe desde un lugar con permisos de escritura)
		log.SetOutput(hub) // por lo menos UI logs
		log.Printf("[boot] InitEarlyLogging error: %v", err)
	} else {
		defer closeFn()
	}

	log.Println("[boot] logger initialized (file + ui hub) ✅")

	// Cargar config para poder purgar logs antiguos (si falla, igual seguimos)
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("[boot] LoadConfig error: %v", err)
		cfg = defaultConfig()
	}
	if err := PurgeOldLogs(cfg.LogRetentionDays); err != nil {
		log.Printf("[boot] PurgeOldLogs error: %v", err)
	}

	// Arrancar UI (no debe reconfigurar el logger)
	runUI(opts, hub)
}
