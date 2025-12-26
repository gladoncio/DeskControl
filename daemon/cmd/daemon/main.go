package main

import (
	"log"
	"os"

	"deskcontrol/daemon/internal/discovery"
	"deskcontrol/daemon/internal/input"
	"deskcontrol/daemon/internal/ws"
)

func main() {
	log.Println("DeskControl daemon starting...")

	wsPort := 54545
	udpPort := 54546

	name, _ := os.Hostname()
	if name == "" {
		name = "DeskControl-PC"
	}

	// UDP discovery responder (para que el móvil lo encuentre sin IP)
	go discovery.StartUDP(name, wsPort, udpPort)

	driver := input.New()
	ws.Start(":54545", driver)
}
