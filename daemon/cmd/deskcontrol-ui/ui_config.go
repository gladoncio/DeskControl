package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"deskcontrol/daemon/internal/startup"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/crypto/bcrypt"
)

func showPairQRDialog(title string, png []byte, payload string, w fyne.Window) {
	res := fyne.NewStaticResource("pair.png", png)
	img := canvas.NewImageFromResource(res)
	img.FillMode = canvas.ImageFillContain
	img.SetMinSize(fyne.NewSize(320, 320))

	entry := widget.NewMultiLineEntry()
	entry.SetText(payload)
	entry.Disable()

	btnCopy := widget.NewButton("Copiar", func() {
		w.Clipboard().SetContent(payload)
	})

	content := container.NewVBox(
		widget.NewLabelWithStyle("Escanea para emparejar", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		container.NewCenter(img),
		widget.NewLabel("Texto (por si el QR falla):"),
		entry,
		btnCopy,
	)

	dialog.NewCustom(title, "Cerrar", content, w).Show()
}

func buildConfigTab(appRunName string, w fyne.Window) fyne.CanvasObject {
	// ---- Load current config ----
	cfg, err := LoadConfig()
	if err != nil {
		log.Printf("[config] LoadConfig error: %v", err)
		cfg = defaultConfig()
	}

	// ---- Autostart status ----
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

	// ✅ FIX: declarar primero y luego asignar
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

	btnRestart := widget.NewButton("Reiniciar app", func() {
		dialog.ShowConfirm("Reiniciar", "¿Reiniciar DeskControl ahora?", func(ok bool) {
			if !ok {
				return
			}
			if err := restartSelf(); err != nil {
				dialog.ShowError(err, w)
			}
		}, w)
	})

	// ---- Networking ----
	entryListenIP := widget.NewEntry()
	entryListenIP.SetPlaceHolder("0.0.0.0 (todas las interfaces)")
	entryListenIP.SetText(strings.TrimSpace(cfg.ListenIP))

	entryWSPort := widget.NewEntry()
	entryWSPort.SetText(strconv.Itoa(cfg.WSPort))

	entryUDPPort := widget.NewEntry()
	entryUDPPort.SetText(strconv.Itoa(cfg.UDPPort))

	checkTLS := widget.NewCheck("Cifrar tráfico (TLS / wss) — si está activado, ws:// NO conectará", func(bool) {})
	checkTLS.SetChecked(cfg.EncryptTrafficTLS)

	entryCert := widget.NewEntry()
	entryCert.SetPlaceHolder("Ruta cert.pem (si TLS)")
	entryCert.SetText(cfg.TLSCertPath)

	entryKey := widget.NewEntry()
	entryKey.SetPlaceHolder("Ruta key.pem (si TLS)")
	entryKey.SetText(cfg.TLSKeyPath)

	// ---- Pairing token ----
	tokenLabel := widget.NewLabel("")
	refreshTokenLabel := func() {
		if strings.TrimSpace(cfg.Token) == "" {
			tokenLabel.SetText("Token: (no generado)")
		} else {
			tokenLabel.SetText("Token: " + cfg.Token)
		}
	}
	refreshTokenLabel()

	checkRequireToken := widget.NewCheck("Requerir token para conectar", func(v bool) {
		cfg.RequireToken = v
		_ = SaveConfig(cfg)
	})
	checkRequireToken.SetChecked(cfg.RequireToken)

	// ✅ NUEVO: Solo mostrar QR (no regenera token)
	btnShowQR := widget.NewButton("Ver QR actual", func() {
		if cfg.EncryptTrafficTLS && strings.TrimSpace(cfg.Token) == "" {
			dialog.ShowInformation("No hay token",
				"Primero genera un token para emparejar.\n\nDespués reinicia DeskControl para que el daemon use el token actualizado.",
				w,
			)
			return
		}

		png, payload, err := tokenQRPNGForConfig(cfg)
		if err != nil {
			dialog.ShowError(err, w)
			return
		}
		showPairQRDialog("QR de Emparejamiento", png, payload, w)
	})

	// ✅ MEJORADO: Genera + QR + alerta reinicio (y ofrece reiniciar)
	btnGenToken := widget.NewButton("Generar token + QR", func() {
		dialog.ShowConfirm("Regenerar token",
			"Esto cambiará el token de emparejamiento.\n\nLuego DEBES reiniciar DeskControl para que el daemon aplique el nuevo token.\n\n¿Deseas continuar?",
			func(ok bool) {
				if !ok {
					return
				}

				tok, err := generateTokenURLSafe(32)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}

				cfg.Token = tok
				cfg.RequireToken = true
				checkRequireToken.SetChecked(true)

				if err := SaveConfig(cfg); err != nil {
					dialog.ShowError(err, w)
					return
				}
				refreshTokenLabel()

				png, payload, err := tokenQRPNGForConfig(cfg)
				if err != nil {
					dialog.ShowError(err, w)
					return
				}
				showPairQRDialog("Nuevo QR (token actualizado)", png, payload, w)

				dialog.ShowConfirm("Reinicio requerido",
					"Token actualizado ✅\n\nPara que el daemon use el nuevo token debes reiniciar DeskControl.\n\n¿Reiniciar ahora?",
					func(ok2 bool) {
						if !ok2 {
							return
						}
						if err := restartSelf(); err != nil {
							dialog.ShowError(err, w)
						}
					}, w,
				)
			},
			w,
		)
	})

	// ---- Account (only TLS) ----
	checkRequireAccount := widget.NewCheck("Requerir cuenta (Basic Auth) — SOLO con TLS", func(v bool) {
		cfg.RequireAccount = v
		_ = SaveConfig(cfg)
	})
	checkRequireAccount.SetChecked(cfg.RequireAccount)

	btnCreateAccount := widget.NewButton("Crear/Actualizar cuenta", func() {
		if !cfg.EncryptTrafficTLS {
			dialog.ShowInformation("TLS requerido", "Activa TLS para usar cuentas.", w)
			return
		}

		u := widget.NewEntry()
		u.SetPlaceHolder("usuario")
		u.SetText(cfg.Username)

		p1 := widget.NewPasswordEntry()
		p1.SetPlaceHolder("contraseña")
		p2 := widget.NewPasswordEntry()
		p2.SetPlaceHolder("repetir contraseña")

		form := widget.NewForm(
			widget.NewFormItem("Usuario", u),
			widget.NewFormItem("Contraseña", p1),
			widget.NewFormItem("Confirmar", p2),
		)

		d := dialog.NewCustomConfirm("Cuenta", "Guardar", "Cancelar", form, func(ok bool) {
			if !ok {
				return
			}
			user := strings.TrimSpace(u.Text)
			if user == "" {
				dialog.ShowError(fmt.Errorf("usuario requerido"), w)
				return
			}
			if p1.Text == "" || p2.Text == "" {
				dialog.ShowError(fmt.Errorf("contraseña requerida"), w)
				return
			}
			if p1.Text != p2.Text {
				dialog.ShowError(fmt.Errorf("las contraseñas no coinciden"), w)
				return
			}

			hash, err := bcrypt.GenerateFromPassword([]byte(p1.Text), bcrypt.DefaultCost)
			if err != nil {
				dialog.ShowError(err, w)
				return
			}

			cfg.Username = user
			cfg.PasswordHash = string(hash)
			cfg.RequireAccount = true
			checkRequireAccount.SetChecked(true)

			if err := SaveConfig(cfg); err != nil {
				dialog.ShowError(err, w)
				return
			}

			dialog.ShowConfirm("Cuenta guardada",
				"Cuenta guardada ✅\n\nDebes reiniciar DeskControl para aplicar.\n\n¿Reiniciar ahora?",
				func(ok2 bool) {
					if !ok2 {
						return
					}
					if err := restartSelf(); err != nil {
						dialog.ShowError(err, w)
					}
				}, w)
		}, w)
		d.Resize(fyne.NewSize(420, 260))
		d.Show()
	})

	// ---- Logs ----
	entryRetention := widget.NewEntry()
	entryRetention.SetText(strconv.Itoa(cfg.LogRetentionDays))

	btnPurge := widget.NewButton("Purgar logs antiguos ahora", func() {
		if err := PurgeOldLogs(cfg.LogRetentionDays); err != nil {
			dialog.ShowError(err, w)
			return
		}
		dialog.ShowInformation("Logs", fmt.Sprintf("Purgado listo (retención: %d días)", cfg.LogRetentionDays), w)
	})

	btnDeleteAllLogs := widget.NewButton("Borrar TODOS los logs", func() {
		dialog.ShowConfirm("Confirmar", "¿Seguro que deseas borrar todos los logs?", func(ok bool) {
			if !ok {
				return
			}
			if err := DeleteAllLogs(); err != nil {
				dialog.ShowError(err, w)
				return
			}
			dialog.ShowInformation("Logs", "Logs borrados ✅", w)
		}, w)
	})

	// ---- Save ----
	saveCfg := func() {
		ncfg := cfg

		ncfg.ListenIP = strings.TrimSpace(entryListenIP.Text)
		if ncfg.ListenIP == "" {
			ncfg.ListenIP = "0.0.0.0"
		}
		if n, err := strconv.Atoi(strings.TrimSpace(entryWSPort.Text)); err == nil {
			ncfg.WSPort = n
		}
		if n, err := strconv.Atoi(strings.TrimSpace(entryUDPPort.Text)); err == nil {
			ncfg.UDPPort = n
		}

		ncfg.EncryptTrafficTLS = checkTLS.Checked
		ncfg.TLSCertPath = strings.TrimSpace(entryCert.Text)
		ncfg.TLSKeyPath = strings.TrimSpace(entryKey.Text)

		ncfg.RequireToken = checkRequireToken.Checked
		ncfg.RequireAccount = checkRequireAccount.Checked

		if n, err := strconv.Atoi(strings.TrimSpace(entryRetention.Text)); err == nil {
			ncfg.LogRetentionDays = n
		}

		// regla: si TLS se apaga, no permitimos cuentas
		if !ncfg.EncryptTrafficTLS {
			ncfg.RequireAccount = false
		}

		if err := SaveConfig(ncfg); err != nil {
			dialog.ShowError(err, w)
			return
		}

		cfg = ncfg
		refreshTokenLabel()
		dialog.ShowConfirm("Configuración",
			"Guardado ✅\n\nPara aplicar red/TLS/auth debes reiniciar DeskControl.\n\n¿Reiniciar ahora?",
			func(ok bool) {
				if !ok {
					return
				}
				if err := restartSelf(); err != nil {
					dialog.ShowError(err, w)
				}
			}, w)
	}

	btnSave := widget.NewButton("Guardar configuración", saveCfg)

	// TLS dependent controls
	refreshTLSDependent := func() {
		if checkTLS.Checked {
			btnCreateAccount.Enable()
			checkRequireAccount.Enable()
			entryCert.Enable()
			entryKey.Enable()
		} else {
			btnCreateAccount.Disable()
			checkRequireAccount.Disable()
			checkRequireAccount.SetChecked(false)
			entryCert.Disable()
			entryKey.Disable()
		}
	}
	checkTLS.OnChanged = func(bool) { refreshTLSDependent() }
	refreshTLSDependent()

	// ---- Layout (scroll fijo) ----
	content := container.NewVBox(
		widget.NewLabelWithStyle("Configuración", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),

		widget.NewLabelWithStyle("Inicio", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		checkAutostart,
		statusLabel,
		container.NewHBox(btnRestart),
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Red (escucha)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("IP de escucha", entryListenIP),
			widget.NewFormItem("Puerto WebSocket", entryWSPort),
			widget.NewFormItem("Puerto UDP (discovery)", entryUDPPort),
		),
		checkTLS,
		widget.NewForm(
			widget.NewFormItem("TLS cert (pem)", entryCert),
			widget.NewFormItem("TLS key (pem)", entryKey),
		),
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Emparejamiento (token)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		tokenLabel,
		container.NewHBox(btnShowQR, btnGenToken, checkRequireToken),
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Cuenta (solo TLS)", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		checkRequireAccount,
		btnCreateAccount,
		widget.NewSeparator(),

		widget.NewLabelWithStyle("Logs", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
		widget.NewForm(
			widget.NewFormItem("Borrar logs después de (días)", entryRetention),
		),
		container.NewHBox(btnPurge, btnDeleteAllLogs),
		widget.NewSeparator(),

		btnSave,
	)

	sc := container.NewVScroll(content)
	sc.SetMinSize(fyne.NewSize(0, 540))
	return sc
}
