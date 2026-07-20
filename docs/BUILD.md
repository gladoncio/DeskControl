# DeskControl — Build Guide

Guía de compilación para todas las plataformas soportadas.

## Project structure

```
DeskControl/
├── daemon/                    # Go module — desktop daemon + GUI
│   ├── cmd/deskcontrol-ui/    # Main app (Fyne UI + core)
│   ├── internal/              # Input drivers, WS server, discovery...
│   ├── go.mod / go.sum
│   └── build/                 # Build output
├── app/                       # Flutter mobile app
│   ├── lib/                   # Dart source code
│   ├── android/               # Android platform
│   ├── ios/                   # iOS platform
│   ├── pubspec.yaml
│   └── ...
├── scripts/                   # Build scripts
│   ├── build-linux.sh
│   ├── build-windows.sh
│   └── build-apk.sh
├── docs/
│   └── BUILD.md               # This file
├── screenshots/               # Project screenshots
└── mobile/                    # Legacy (ignorar)
```

---

## Requirements

### Go daemon (Linux / Windows)

| Dependency | Linux | Windows (native) | Windows (cross desde Linux) |
|------------|-------|------------------|------------------------------|
| Go ≥ 1.24 | `apt install golang` | [Descargar](https://go.dev/dl/) | `apt install golang` |
| GCC (CGO) | `apt install gcc` | MSYS2 + MinGW-w64 | Docker (fyne-cross) |
| X11 dev (Linux only) | `apt install libx11-dev libxtst-dev` | — | — |
| Docker | — | — | `apt install docker.io` |

#### Linux: dependencias detalladas

```bash
# Debian / Ubuntu
sudo apt install golang gcc libx11-dev libxtst-dev
```

```bash
# Arch Linux
sudo pacman -S go base-devel libx11 libxtst
```

```bash
# Fedora
sudo dnf install golang gcc libX11-devel libXtst-devel
```

#### Windows (native): dependencias detalladas

1. Instalar [Go](https://go.dev/dl/) (≥ 1.24)
2. Instalar [MSYS2](https://www.msys2.org/) con MinGW-w64 (para GCC y CGO)
   ```bash
   # Dentro de MSYS2
   pacman -S mingw-w64-ucrt-x86_64-gcc
   ```
3. Agregar `C:\msys64\ucrt64\bin` al `PATH`

### Flutter app (Android / iOS)

| Dependency | Android | iOS |
|------------|---------|-----|
| Flutter SDK (stable) | ✅ | ✅ |
| Android SDK (via Android Studio o CLI) | ✅ | — |
| JDK 17+ | ✅ (bundled con Android Studio) | — |
| Xcode (macOS) | — | ✅ |
| CocoaPods (macOS) | — | ✅ |

#### Android: dependencias detalladas

```bash
# 1. Instalar Flutter SDK
git clone https://github.com/flutter/flutter.git -b stable ~/flutter
export PATH="$PATH:$HOME/flutter/bin"

# 2. Verificar dependencias
flutter doctor

# 3. Aceptar licencias de Android
flutter doctor --android-licenses
```

#### iOS: dependencias detalladas

> **Nota**: iOS solo puede compilarse en macOS con Xcode instalado.

```bash
# 1. Instalar CocoaPods
sudo gem install cocoapods

# 2. Configurar Flutter
flutter precache --ios

# 3. Verificar
flutter doctor
```

---

## Quick start

### 1) Linux daemon

```bash
cd daemon
./scripts/build-linux.sh
# o manualmente:
go build -o deskcontrol-daemon ./cmd/deskcontrol-ui
./deskcontrol-daemon          # mostrar ventana
./deskcontrol-daemon -tray    # iniciar en bandeja del sistema
```

### 2) Windows daemon (cross-compile desde Linux)

```bash
./scripts/build-windows.sh
# Output: daemon/fyne-cross/dist/windows-amd64/DeskControl.exe.zip
```

### 3) Windows daemon (native en Windows con MSYS2)

```powershell
cd daemon
$env:CGO_ENABLED="1"
$env:CC="gcc"
$env:CXX="g++"
go build -o build/DeskControl.exe ./cmd/desktopcontrol-ui
```

### 4) Android APK

```bash
./scripts/build-apk.sh
# Output: app/build/app/outputs/flutter-apk/app-release.apk
```

### 5) iOS IPA (solo macOS)

```bash
cd app
flutter build ios --release
# Output: build/ios/iphoneos/Runner.app
# Para IPA: Product → Archive en Xcode
```

---

## Build details

### Linux — go build (native)

```bash
cd daemon
go build -o deskcontrol-daemon ./cmd/deskcontrol-ui
```

Flags:

- `-o deskcontrol-daemon` — output binary name
- `./cmd/deskcontrol-ui` — main package (the only one with `func main()`)

Run:

```bash
./deskcontrol-daemon          # mostrar ventana
./deskcontrol-daemon -tray    # iniciar en bandeja del sistema, oculto
```

#### Linux: compilación con debug

```bash
cd daemon
go build -race -o deskcontrol-daemon-debug ./cmd/deskcontrol-ui
```

#### Linux: cross-compile para ARM (Raspberry Pi)

```bash
cd daemon
GOARCH=arm64 GOOS=linux CGO_ENABLED=1 CC=aarch64-linux-gnu-gcc \
    go build -o deskcontrol-daemon-arm64 ./cmd/deskcontrol-ui
```

> Requiere toolchain: `sudo apt install gcc-aarch64-linux-gnu`

### Windows — fyne-cross (Docker, desde Linux)

Cross-compilation usa [fyne-cross](https://github.com/fyne-io/fyne-cross), que ejecuta una toolchain MinGW-w64 dentro de Docker.

```bash
# Instalar fyne-cross
go install github.com/fyne-io/fyne-cross@latest

# Compilar
fyne-cross windows \
    -app-id com.deskcontrol.daemon \
    -name DeskControl \
    -icon cmd/deskcontrol-ui/assets/tray.png \
    -output DeskControl.exe \
    ./cmd/deskcontrol-ui
```

Output: `fyne-cross/dist/windows-amd64/DeskControl.exe.zip`

### Windows — native build (en Windows con MSYS2)

```powershell
cd daemon
$env:CGO_ENABLED="1"
$env:CC="gcc"
$env:CXX="g++"
go build -o build/DeskControl.exe ./cmd/deskcontrol-ui
```

El binario se genera en `daemon/build/DeskControl.exe`.

### Android APK — flutter build

```bash
cd app
flutter pub get
flutter build apk --release
```

Output: `app/build/app/outputs/flutter-apk/app-release.apk`

#### APK para desarrollo (debug)

```bash
cd app
flutter build apk --debug
# Output: app/build/app/outputs/flutter-apk/app-debug.apk
```

#### APK con split por ABI (tamaño reducido)

```bash
cd app
flutter build apk --split-per-abi
# Output: app/build/app/outputs/flutter-apk/app-armeabi-v7a-release.apk
#         app/build/app/outputs/flutter-apk/app-arm64-v8a-release.apk
#         app/build/app/outputs/flutter-apk/app-x86_64-release.apk
```

#### APK firmado para release

Para distribuir, crear `app/android/key.properties`:

```properties
storePassword=<password>
keyPassword=<password>
keyAlias=upload
storeFile=<path-to-keystore>
```

Luego:

```bash
cd app
flutter build apk --release
```

Ver [Flutter docs — Signing the app](https://docs.flutter.dev/deployment/android#signing-the-app).

### Android App Bundle (AAB) — Google Play

```bash
cd app
flutter build appbundle
# Output: app/build/app/outputs/bundle/release/app-release.aab
```

### iOS IPA (solo macOS)

```bash
cd app
flutter build ios --release --no-codesign
# Output: build/ios/iphoneos/Runner.app
```

Para firmar y generar IPA:

1. Abrir `app/ios/Runner.xcworkspace` en Xcode
2. Configurar team de firma en Signing & Capabilities
3. `Product → Archive → Distribute App`

O via línea de comandos con Fastlane:

```bash
cd app/ios
fastlane match development
fastlane build
```

---

## Scripts de build

El proyecto incluye scripts listos para usar en `scripts/`:

| Script | Propósito | Requisitos |
|--------|-----------|------------|
| `build-linux.sh` | Compila daemon para Linux | Go, GCC, libx11-dev, libxtst-dev |
| `build-windows.sh` | Cross-compila daemon para Windows | Go, Docker, fyne-cross |
| `build-apk.sh` | Compila APK Android | Flutter SDK, Android SDK, JDK |

Ejecutar desde la raíz del proyecto:

```bash
./scripts/build-linux.sh
./scripts/build-windows.sh
./scripts/build-apk.sh
```

---

## Input driver selection

El daemon selecciona automáticamente el mejor driver de entrada al iniciar:

| Driver | Platform | When used |
|--------|----------|-----------|
| **uinput** | Linux | Default. Escribe directamente en el subsistema de entrada del kernel. Crea un dispositivo virtual `/dev/uinput`. |
| **XTest** | Linux | Fallback si `/dev/uinput` no está disponible. Inyecta eventos vía X11 (`libXtst`). |
| **SendInput** | Windows | Siempre (API Win32 nativa). |
| **evdev** | Linux | Captura de teclas solamente (lee `/dev/input/event*`). |
| **D-Bus/AT-SPI** | Linux | Captura de teclas fallback (Wayland a11y bus). |
| **X11** | Linux | Captura de teclas fallback si evdev + D-Bus no están disponibles. |

---

## Ports

| Port | Protocol | Purpose |
|------|----------|---------|
| 54545 | WebSocket | Canal principal de control (ratón, teclado, apps) |
| 54546 | UDP | Descubrimiento / anuncio en LAN |

---

## Protocolo WebSocket

Los mensajes se intercambian como JSON sobre WebSocket en `ws://host:54545/ws` (o `wss://` con TLS).

### Formato de mensaje

```json
{
  "id": "uuid-unico",
  "type": "comando",
  "payload": { ... }
}
```

### Tipos de mensaje

| Tipo | Dirección | Descripción |
|------|-----------|-------------|
| `ping` | App → Daemon | Handshake inicial (`{"app":"deskcontrol","v":1}`) |
| `mouse_move` | App → Daemon | Mover ratón (`dx`, `dy`) |
| `mouse_click` | App → Daemon | Clic (`button`: "left"/"right"/"middle") |
| `mouse_down` / `mouse_up` | App → Daemon | Presionar/soltar botón |
| `mouse_scroll` | App → Daemon | Scroll (`delta`) |
| `key_text` | App → Daemon | Escribir texto (`text`) |
| `key` / `key_down` / `key_up` | App → Daemon | Tecla por nombre |
| `key_vk` / `key_down_vk` / `key_up_vk` | App → Daemon | Tecla por VK (Windows Virtual Key) |
| `hotkey` / `hotkey_vk` | App → Daemon | Atajo de teclado |
| `text_input` | App → Daemon | Entrada de texto (alternativo) |
| `input_key_tap` / `input_key_down` / `input_key_up` | App → Daemon | Tecla por KeySpec (vk, scan, ext) |
| `capture_start` | App → Daemon | Solicita capturar la siguiente tecla pulsada en el PC |
| `capture_result` | Daemon → App | Resultado de la captura (`keySpec`, `modifiers`) |
| `apps_list` | App → Daemon | Solicita lista de ventanas abiertas |
| `apps_result` | Daemon → App | Lista de ventanas (`windows[]`) |
| `app_action` | App → Daemon | Acción sobre ventana (`hwnd`, `action`: "activate"/"minimize"/"maximize"/"close") |

### Autenticación

1. **Token** (cuando TLS está habilitado):
   - Header HTTP: `X-DeskControl-Token: <token>`
   - Query param: `?token=<token>`
   - Header: `Authorization: Bearer <token>`

2. **Usuario/contraseña** (cuando TLS + cuentas están habilitados):
   - Mensaje `auth_login` con `username` y `password`
   - Verificación bcrypt contra SQLite

### Cifrado de payload (legacy)

Cuando el flag `securePayload` está activo, el mensaje JSON se cifra con AES-256-GCM:

```
nonce = 12 bytes aleatorios
key = SHA-256(token)
ciphertext = AES-GCM(nonce, plaintext, aad="deskcontrol-v1")
```

Envelope:

```json
{
  "enc": 1,
  "nonce": "<base64>",
  "data": "<base64>",
  "tag": "<base64>"
}
```

---

## SQLite database

El daemon almacena configuración y usuarios en una base de datos SQLite.

### Ubicación

| Plataforma | Ruta |
|-----------|------|
| Linux | `~/.config/DeskControl/deskcontrol.db` |
| Windows | `%APPDATA%\DeskControl\deskcontrol.db` |

### Tablas

| Tabla | Propósito |
|-------|-----------|
| `settings` | Configuración del daemon (key-value JSON) |
| `users` | Usuarios registrados (username, bcrypt hash, disabled, last_login) |

---

## Troubleshooting

### “function main is undeclared in the main package”

Intentaste compilar la raíz del módulo o `cmd/daemon/` (que es una librería, no un binario). Compila **`cmd/deskcontrol-ui`** en su lugar:

```bash
go build ./cmd/deskcontrol-ui
```

### uinput not available: UI_DEV_CREATE: invalid argument

La API de uinput del kernel ha cambiado. El daemon usa el ioctl `UI_DEV_SETUP` moderno (Linux ≥ 4.12). Si ves esto, asegúrate de que tu kernel esté actualizado.

Para kernels antiguos (< 4.12), puedes forzar el driver XTest:

```bash
DESKCONTROL_INPUT_DRIVER=xtest ./deskcontrol-daemon
```

### Spaces not typed on Linux (XTest driver)

Solucionado en `xtest_linux.go`: `XStringToKeysym(" ")` devuelve `NoSymbol` en X11, así que el daemon usa el código de carácter directamente como keysym.

### Arrow keys trapped in daemon window

Ejecuta el daemon con `-tray` para ocultar la ventana, o asegúrate de que `uinput` esté disponible (inyecta a nivel de kernel, evitando el foco de la ventana).

### Flutter build fails: Gradle sync issues

```bash
cd app
flutter clean
flutter pub get
cd android
./gradlew clean
cd ..
flutter build apk
```

### WebSocket connection refused

- Verifica que el daemon esté corriendo: `./deskcontrol-daemon`
- Verifica que el puerto 54545 esté abierto: `ss -tlnp | grep 54545`
- Verifica el firewall: `sudo ufw status`

### No se encontraron daemons en UDP discovery

- Verifica que el daemon y el móvil estén en la misma red LAN
- Verifica que el puerto UDP 54546 no esté bloqueado
- Conexión manual: abre Ajustes en la app e ingresa la IP manualmente

---

## Output artifacts

| Artifact | Path |
|----------|------|
| Linux binary | `daemon/deskcontrol-daemon` |
| Windows binary (cross) | `daemon/fyne-cross/dist/windows-amd64/DeskControl.exe.zip` |
| Windows binary (native) | `daemon/build/DeskControl.exe` |
| Android APK (release) | `app/build/app/outputs/flutter-apk/app-release.apk` |
| Android APK (debug) | `app/build/app/outputs/flutter-apk/app-debug.apk` |
| Android App Bundle | `app/build/app/outputs/bundle/release/app-release.aab` |
| iOS build | `app/build/ios/iphoneos/Runner.app` |
