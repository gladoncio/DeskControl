package main

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

// buildUsersTab: pestaña dedicada para administración de usuarios.
// Por ahora es “sólo UI” (placeholder). En el siguiente paso la conectamos a SQLite.
func buildUsersTab(w fyne.Window) fyne.CanvasObject {
	info := widget.NewLabel(
		"Usuarios (solo TLS)\n\n" +
			"Meta:\n" +
			"• Crear/editar/desactivar usuarios\n" +
			"• Ver conexiones activas y cortar sesiones\n" +
			"• Solo aplica si TLS está activado y 'Requerir login' está ON\n",
	)

	btnHow := widget.NewButton("¿Qué falta para que funcione?", func() {
		dialog.ShowInformation(
			"Pendiente",
			"En el siguiente paso vamos a:\n\n"+
				"1) Crear una tabla users en SQLite\n"+
				"2) Agregar CRUD aquí (crear/listar/desactivar/borrar)\n"+
				"3) Cambiar el daemon para autenticar por WS (auth_login) usando esa tabla\n"+
				"   (más escalable que BasicAuth HTTP).\n",
			w,
		)
	})

	btnGoConfig := widget.NewButton("Ir a Config (activar TLS / login)", func() {
		dialog.ShowInformation(
			"Tip",
			"Activa TLS y luego habilita 'Requerir login' (lo vamos a mover a Config).\n"+
				"Después creas usuarios aquí.",
			w,
		)
	})

	return container.NewVScroll(
		container.NewVBox(
			widget.NewLabelWithStyle("Usuarios", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
			info,
			container.NewHBox(btnHow, btnGoConfig),
		),
	)
}
