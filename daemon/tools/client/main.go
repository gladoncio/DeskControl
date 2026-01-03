package main

import (
	"flag"
	"fmt"
	"time"

	"deskcontrol/daemon/internal/discovery"
)

func main() {
	name := flag.String("name", "DeskControl-PC", "nombre a anunciar")
	wsPort := flag.Int("ws", 54545, "puerto WebSocket a anunciar")
	udpPort := flag.Int("udp", 54546, "puerto UDP discovery (escucha)")
	listenIP := flag.String("listen", "0.0.0.0", "IP local a la que se bindea el UDP (0.0.0.0 = todas)")
	flag.Parse()

	fmt.Printf("DeskControl discovery responder\n")
	fmt.Printf("  name=%s ws=%d udp=%d listen=%s\n", *name, *wsPort, *udpPort, *listenIP)
	fmt.Printf("  (Ctrl+C para salir)\n\n")

	// Responder a "discover" por UDP con "announce"
	go discovery.StartUDP(*name, *wsPort, *udpPort, *listenIP, false)

	// Bloquea para mantener vivo el proceso
	for {
		time.Sleep(1 * time.Hour)
	}
}
