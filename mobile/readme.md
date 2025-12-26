# DeskControl (Mobile Flutter App) — Build Guide + Release APK 1.1

This document describes how to build the **DeskControl mobile app** (Flutter) and produce a **release APK** for distribution.

---

## Project location (repo layout)

- `mobile/deskcontrol_app/`
  - `pubspec.yaml`
  - `android/`
  - `ios/`
  - `lib/`
  - `assets/` *(recommended for app icon and shared assets)*

---

## Requirements

### Windows (your current dev environment)
- Flutter SDK (stable) + `flutter doctor` OK
- Android SDK + Platform tools (ADB)
- Java (JDK) required for Android builds (Flutter Doctor will guide)
- For signing release builds: keytool (comes with JDK)

---

## One-time setup

### 1) Verify toolchain
From the Flutter project folder:

```powershell
cd .\mobile\deskcontrol_app
flutter doctor
```

Fix any Android toolchain issues it reports (SDK licenses, missing cmdline-tools, etc).

---

## App name (DeskControl)

### Android label
Set `android:label="DeskControl"` in:

- `android/app/src/main/AndroidManifest.xml`

If it uses a string resource, update:

- `android/app/src/main/res/values/strings.xml`
  - `<string name="app_name">DeskControl</string>`

### iOS display name
Edit `ios/Runner/Info.plist`:

- `CFBundleDisplayName` = `DeskControl`

---

## App icon (same as desktop)

### Recommended: use flutter_launcher_icons

1) Put a **square PNG** here:

- `assets/icon.png` (1024x1024 recommended)

2) In `pubspec.yaml`, add:

```yaml
dev_dependencies:
  flutter_launcher_icons: ^0.14.4

flutter_launcher_icons:
  android: true
  ios: true
  image_path: "assets/icon.png"
```

3) Run:

```powershell
flutter pub get
dart run flutter_launcher_icons
```

---

## Versioning for Release 1.1

In `pubspec.yaml`, set the version line to:

```yaml
version: 1.1.0+110
```

- `1.1.0` = human version
- `+110` = build number (integer). Use any scheme you like, but it must increase for updates.

---

## Build (Debug)

```powershell
cd .\mobile\deskcontrol_app
flutter pub get
flutter run
```

---

## Build APK (Release) — for direct download

### A) Quick unsigned-ish (NOT recommended for sharing)
Flutter can build a release APK, but for real distribution you should sign it.
If you just want a local test:

```powershell
flutter build apk --release
```

Output:
- `build\app\outputs\flutter-apk\app-release.apk`

### B) Proper signed release APK (recommended)

#### 1) Generate a keystore (one-time)
From the Flutter project root:

```powershell
cd .\mobile\deskcontrol_app
keytool -genkeypair -v -storetype JKS -keyalg RSA -keysize 2048 -validity 10000 ^
  -alias deskcontrol ^
  -keystore android\app\upload-keystore.jks
```

Set a strong password and keep it safe.

#### 2) Create `android/key.properties` (DO NOT COMMIT)
Create file: `android/key.properties`

Example:

```properties
storePassword=YOUR_STORE_PASSWORD
keyPassword=YOUR_KEY_PASSWORD
keyAlias=deskcontrol
storeFile=upload-keystore.jks
```

#### 3) Configure signing in `android/app/build.gradle`
Open `android/app/build.gradle` and ensure it loads `key.properties`
and uses it in `signingConfigs { release { ... } }`.

Typical snippet (place near the top):

```gradle
def keystoreProperties = new Properties()
def keystorePropertiesFile = rootProject.file('key.properties')
if (keystorePropertiesFile.exists()) {
    keystoreProperties.load(new FileInputStream(keystorePropertiesFile))
}
```

And inside `android { ... }`:

```gradle
signingConfigs {
    release {
        keyAlias keystoreProperties['keyAlias']
        keyPassword keystoreProperties['keyPassword']
        storeFile file(keystoreProperties['storeFile'])
        storePassword keystoreProperties['storePassword']
    }
}

buildTypes {
    release {
        signingConfig signingConfigs.release
        // keep default minify/shrink disabled unless you want it
    }
}
```

> NOTE: Some new Flutter templates already include this; just update paths and aliases.

#### 4) Build signed APK
```powershell
flutter clean
flutter pub get
flutter build apk --release
```

Output:
- `build\app\outputs\flutter-apk\app-release.apk`

This APK should be installable on devices (if “install unknown apps” is allowed).

---

## Build App Bundle (AAB) (optional, Play Store friendly)

```powershell
flutter build appbundle --release
```

Output:
- `build\app\outputs\bundle\release\app-release.aab`

---

## Output artifacts

- Release APK:
  - `mobile/deskcontrol_app/build/app/outputs/flutter-apk/app-release.apk`
- Release AAB:
  - `mobile/deskcontrol_app/build/app/outputs/bundle/release/app-release.aab`
