package main

import (
	"log"

	"deskcontrol/daemon/internal/startup"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func buildConfigTab(appRunName string, w fyne.Window) fyne.CanvasObject {
	enabled, existingCmd, err := startup.IsEnabled(appRunName)
	if err != nil {
		log.Printf("[config] startup.IsEnabled error: %v", err)
		enabled = false
		existingCmd = ""
	}

	statusLabel := widget.NewLabel("")
	setStatus := func() {
		if enabled {
			statusLabel.SetText("Autostart: ACTIVADO (HKCU Run)\n" + existingCmd)
		} else {
			statusLabel.SetText("Autostart: DESACTIVADO")
		}
	}
	setStatus()

	var checkAutostart *widget.Check
	checkAutostart = widget.NewCheck("Iniciar con Windows (modo tray)", func(v bool) {
		args := "--tray"
		if err := startup.SetEnabled(appRunName, v, args); err != nil {
			log.Printf("[config] SetEnabled error: %v", err)
			enabled = !v
			checkAutostart.SetChecked(enabled)
			setStatus()
			return
		}

		enabled2, cmd2, err2 := startup.IsEnabled(appRunName)
		if err2 != nil {
			log.Printf("[config] IsEnabled error: %v", err2)
		}
		enabled = enabled2
		existingCmd = cmd2
		setStatus()
	})
	checkAutostart.SetChecked(enabled)

	btnHideToTray := widget.NewButton("Ocultar al tray ahora", func() {
		w.Hide()
	})

	return container.NewVBox(
		widget.NewLabelWithStyle("Configuración", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		checkAutostart,
		statusLabel,
		widget.NewSeparator(),
		btnHideToTray,
	)
}
