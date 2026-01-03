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

type HubIface interface {
	Snapshot() []string
	Subscribe(n int) (<-chan string, func())
	Clear()
}

func startCoreFromConfig() {
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("[ui] LoadConfig error: %v (usando default)", err)
		cfg = defaultConfig()
	}

	name, _ := os.Hostname()
	if name == "" {
		name = "DeskControl-PC"
	}

	// ✅ StartUDP ahora requiere 5 args: (name, wsPort, udpPort, bindIP, tlsOn)
	go discovery.StartUDP(name, cfg.WSPort, cfg.UDPPort, cfg.ListenIP, cfg.EncryptTrafficTLS)

	// WS
	driver := input.New()

	addr := fmt.Sprintf(":%d", cfg.WSPort)
	if ip := cfg.ListenIP; ip != "" && ip != "0.0.0.0" {
		addr = fmt.Sprintf("%s:%d", ip, cfg.WSPort)
	}

	sec := ws.SecurityConfig{
		RequireTLS:     cfg.EncryptTrafficTLS,
		CertPath:       cfg.TLSCertPath,
		KeyPath:        cfg.TLSKeyPath,
		RequireToken:   cfg.RequireToken,
		Token:          cfg.Token,
		RequireAccount: cfg.RequireAccount,
	}

	log.Printf("[core] running WS=%s UDP=%d (bind=%s) tls=%v token=%v account=%v",
		addr, cfg.UDPPort, cfg.ListenIP, cfg.EncryptTrafficTLS, cfg.RequireToken, cfg.RequireAccount)

	go ws.Start(addr, driver, sec)
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

	// ✅ Core (WS + UDP) antes de mostrar UI
	log.Printf("[ui] starting core…")
	startCoreFromConfig()

	logsTab := buildLogsTab(a, w, hub, state, opts.MaxUILines, opts.Tick)
	configTab := buildConfigTab(opts.AppRunName, w)

	// ✅ NUEVA pestaña dedicada
	usersTab := buildUsersTab(w)

	tabs := container.NewAppTabs(
		container.NewTabItem("Logs", logsTab),
		container.NewTabItem("Config", configTab),
		container.NewTabItem("Usuarios", usersTab),
	)
	w.SetContent(tabs)

	// Tray
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

		if iconRes != nil {
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
