package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"deskcontrol/daemon/internal/discovery"
	"deskcontrol/daemon/internal/input"
	"deskcontrol/daemon/internal/ws"

	"github.com/getlantern/systray"
)

func startDaemon(wsPort, udpPort int) {
	name, _ := os.Hostname()
	if name == "" {
		name = "DeskControl-PC"
	}

	go discovery.StartUDP(name, wsPort, udpPort)

	driver := input.New()
	go ws.Start(fmt.Sprintf(":%d", wsPort), driver)

	log.Printf("Daemon running: WS=%d UDP=%d\n", wsPort, udpPort)
}

func main() {
	tray := flag.Bool("tray", false, "run in system tray")
	wsPort := flag.Int("ws", 54545, "websocket port")
	udpPort := flag.Int("udp", 54546, "udp discovery port")
	flag.Parse()

	if *tray {
		systray.Run(func() {
			systray.SetTitle("DeskControl")
			systray.SetTooltip("DeskControl daemon")

			startDaemon(*wsPort, *udpPort)

			mStatus := systray.AddMenuItem("Running (WS:54545 / UDP:54546)", "Status")
			mStatus.Disable()

			systray.AddSeparator()
			mExit := systray.AddMenuItem("Exit", "Close DeskControl")

			go func() {
				<-mExit.ClickedCh
				systray.Quit()
			}()
		}, func() {})
		return
	}

	startDaemon(*wsPort, *udpPort)
	select {}
}
