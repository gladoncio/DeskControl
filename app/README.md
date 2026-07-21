# DeskControl — App móvil

App Flutter para el control remoto del PC. Se conecta al daemon DeskControl vía WebSocket y permite usar el móvil como ratón, teclado y control de aplicaciones/sonido.

## Requisitos

- Flutter SDK (stable)
- Android SDK (para build APK)
- Xcode + CocoaPods (para build iOS, solo macOS)

## Compilación

```bash
flutter pub get
flutter build apk --release        # Android
flutter build ios --release         # iOS (solo macOS)
```

Ver [`docs/BUILD.md`](../docs/BUILD.md) para más detalles.
