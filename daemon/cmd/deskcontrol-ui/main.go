package main

import (
	"flag"
	"log"
	"time"

	"deskcontrol/daemon/internal/loghub"
)

func main() {
	wsPort := flag.Int("ws", 54545, "websocket port")
	udpPort := flag.Int("udp", 54546, "udp discovery port")
	startInTray := flag.Bool("tray", false, "start minimized to system tray (hide window)")
	flag.Parse()

	hub := loghub.New(4000)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)
	log.SetOutput(hub)

	startCore(*wsPort, *udpPort)

	runUI(UIOpts{
		StartInTray: *startInTray,
		AppRunName:  "DeskControl",
		MaxUILines:  100,
		Tick:        200 * time.Millisecond,
	}, hub)
}
