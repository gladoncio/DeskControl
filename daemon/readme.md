# DeskControl (Daemon + Desktop UI) — Build Guide

This document describes how to build the **DeskControl Windows daemon + desktop UI** (Go + Fyne) from a fresh clone.

---

## Project location (repo layout)

Typical layout (your current repo):

- `daemon/` (Go module, desktop UI + daemon)
  - `go.mod`
  - `cmd/daemon/`
  - `cmd/deskcontrol-ui/`
  - `internal/`
  - `build/` *(output folder, not committed)*

---

## Requirements

### Windows (recommended for your current setup)
- **Windows 10/11 x64**
- **Go** installed (your repo uses Go `1.25.5` in `go.mod`)
- **MSYS2** with MinGW-w64 toolchain (needed because Fyne uses CGO on Windows)
  - `gcc.exe` and `g++.exe` from `C:\msys64\mingw64\bin`

### Optional (recommended)
- Git
- A proper app icon PNG:
  - `daemon/cmd/deskcontrol-ui/assets/app.png` (512x512 or 1024x1024)
  - `daemon/cmd/deskcontrol-ui/assets/tray.png` (16–64px for tray)

---

## One-time setup (Windows)

### 1) Install MSYS2 + MinGW GCC
If you already installed MSYS2, ensure gcc exists:

- `C:\msys64\mingw64\bin\gcc.exe`
- `C:\msys64\mingw64\bin\g++.exe`

You can add MinGW to PATH for the current terminal (recommended):

```powershell
$env:Path = "C:\msys64\mingw64\bin;$env:Path"
```

### 2) Enable CGO and set compiler
For the current PowerShell session:

```powershell
$env:CGO_ENABLED="1"
$env:CC="gcc"
$env:CXX="g++"
```

(If you didn't add MinGW to PATH, use full paths instead)

```powershell
$env:CC="C:\msys64\mingw64\bin\gcc.exe"
$env:CXX="C:\msys64\mingw64\bin\g++.exe"
```

---

## Build (Debug)

From `daemon/`:

```powershell
cd .\daemon
go mod download

# Builds with console (good for debugging)
go build -o .\build\DeskControl-debug.exe .\cmd\deskcontrol-ui
```

Run it:

```powershell
.\build\DeskControl-debug.exe
```

---

## Build (Release — no console window)

From `daemon/`:

```powershell
cd .\daemon
go mod download

# No console window
go build -o .\build\DeskControl.exe -ldflags "-H=windowsgui" .\cmd\deskcontrol-ui
```

---

## Build (Release WITH embedded EXE icon)

### Recommended approach: use the *new* Fyne tools CLI

**Why:** `go build` cannot embed Windows icons by itself. Packaging via Fyne adds app metadata and icon resources.

### 1) Install the correct CLI (new tools)
Run:

```powershell
go install fyne.io/tools/cmd/fyne@latest
```

Ensure Go bin is in PATH (current session):

```powershell
$env:Path += ";$env:USERPROFILE\go\bin"
```

Verify:

```powershell
fyne version
```

> If you still see `fyne cli version: v1.x` from the old CLI, remove/rename the old `fyne.exe` in `C:\Users\<you>\go\bin` and reinstall.
> The correct one is installed from `fyne.io/tools/cmd/fyne`.

### 2) Package from the UI folder (avoids module root confusion)

```powershell
cd .\daemon\cmd\deskcontrol-ui
fyne package -os windows -name DeskControl -icon .\assets\app.png
```

This generates `DeskControl.exe` in the current directory. Move it to build output:

```powershell
Move-Item .\DeskControl.exe ..\..\build\DeskControl.exe -Force
```

> If you don't have `assets/app.png` yet, you can temporarily use `assets/tray.png`,
> but it may look blurry because it’s usually small.

---

## Notes for Linux later

When you move to Linux:
- Install Go
- Install system dependencies required by Fyne (varies by distro; typically X11/GL, dev headers)
- Build normally with `go build` or use:
  - `fyne package -os linux ...`

Keep Windows-specific source files under `//go:build windows` (you already do this in `internal/input/*_windows.go`).

---

## Troubleshooting

### “function main is undeclared in the main package”
That means you tried to build/package the module root (which has no `main()`).
Build/package **`cmd/deskcontrol-ui`** instead.

### “runtime/cgo: ... cgo.exe exit status 2”
Usually means missing compiler toolchain.
Confirm MinGW GCC is installed and `CC/CXX` are set to it.

### Systray icon errors (unknown format / tray not ready)
- Use PNG bytes, not ICO, for some tray libraries.
- Ensure your tray icon is a valid PNG.
- Some systray implementations need initialization before setting icon.

---

## Output artifacts

- `daemon/build/DeskControl-debug.exe`
- `daemon/build/DeskControl.exe` (Windows GUI, release)
