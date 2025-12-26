package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"deskcontrol/daemon/internal/discovery"
	"deskcontrol/daemon/internal/input"
	"deskcontrol/daemon/internal/ws"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
)

type UIOpts struct {
	StartInTray bool
	AppRunName  string
	MaxUILines  int
	Tick        time.Duration
}

type UIState struct {
	ShowUI bool
}

func startCore(wsPort, udpPort int) {
	name, _ := os.Hostname()
	if name == "" {
		name = "DeskControl-PC"
	}

	go discovery.StartUDP(name, wsPort, udpPort)

	driver := input.New()
	go ws.Start(fmt.Sprintf(":%d", wsPort), driver)

	log.Printf("[core] running WS=%d UDP=%d", wsPort, udpPort)
}

type HubIface interface {
	Snapshot() []string
	Subscribe(n int) (<-chan string, func())
	Clear()
}

func runUI(opts UIOpts, hub HubIface) {
	state := &UIState{ShowUI: true}

	a := app.New()
	w := a.NewWindow("DeskControl")
	w.Resize(fyne.NewSize(950, 650))

	// Icon resource (si existe assets.go/trayPng)
	var iconRes fyne.Resource
	if trayPng != nil && len(trayPng) > 0 {
		iconRes = fyne.NewStaticResource("tray.png", trayPng)
		a.SetIcon(iconRes)
		w.SetIcon(iconRes)
	}

	logsTab := buildLogsTab(a, w, hub, state, opts.MaxUILines, opts.Tick)
	configTab := buildConfigTab(opts.AppRunName, w)

	tabs := container.NewAppTabs(
		container.NewTabItem("Logs", logsTab),
		container.NewTabItem("Config", configTab),
	)
	w.SetContent(tabs)

	// Tray menu + cerrar => ocultar
	if desk, ok := a.(desktop.App); ok {
		menuShow := fyne.NewMenuItem("Abrir", func() {
			state.ShowUI = true
			w.Show()
			w.RequestFocus()
		})
		menuHide := fyne.NewMenuItem("Ocultar", func() {
			state.ShowUI = false
			w.Hide()
		})
		menuExit := fyne.NewMenuItem("Salir", func() {
			a.Quit()
		})
		desk.SetSystemTrayMenu(fyne.NewMenu("DeskControl", menuShow, menuHide, menuExit))

		// ✅ IMPORTANTE: setear el icono del tray cuando la app ya arrancó,
		// así no aparece "tray not ready yet"
		if iconRes != nil {
			// Fyne Lifecycle existe en v2
			a.Lifecycle().SetOnStarted(func() {
				desk.SetSystemTrayIcon(iconRes)
			})
		}
	}

	w.SetCloseIntercept(func() {
		state.ShowUI = false
		w.Hide()
	})

	log.Printf("[ui] started")

	if opts.StartInTray {
		state.ShowUI = false
		w.Hide()
		a.Run()
		return
	}

	state.ShowUI = true
	w.ShowAndRun()
}
