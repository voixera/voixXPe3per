# voiXPe3per

voiXPe3per adalah starter project untuk mirroring layar Android/iOS ke desktop Windows. Mode default memakai relay WebSocket publik, jadi mobile device dan desktop tidak harus berada di jaringan yang sama. Desktop menampilkan QR pairing, browser mobile membuka halaman Vercel, lalu user cukup klik `Izinkan pairing`.

## Struktur

```text
phone-mirror/
  desktop/   Wails v2 + Go backend + React/TypeScript/Tailwind UI
  android/   Kotlin Android app, ZXing QR scanner, MediaProjection, H264 encoder
  ios/       Optional SwiftUI native client untuk pengembangan lanjutan
  web/       Vercel pairing page tanpa install aplikasi
  shared/    Protocol docs, JSON schema, shared model contracts
  scripts/   Build helpers
```

## Desktop

### Install Windows via PowerShell

Buka PowerShell biasa, lalu jalankan satu perintah ini:

```powershell
irm https://raw.githubusercontent.com/voixera/voixXPe3per/main/scripts/install-windows.ps1 | iex
```

Link script: `https://raw.githubusercontent.com/voixera/voixXPe3per/main/scripts/install-windows.ps1`

Script memasang Git, Go, Node.js LTS, WinLibs GCC, dan Wails bila belum ada. Lalu script clone source, build EXE, dan memasang perintah `voixpe3per` global. Membutuhkan `winget`, koneksi internet, dan ruang disk untuk build.

Prerequisite:

- Go 1.22+
- Node.js 20+
- Wails v2 CLI
- FFmpeg tersedia di PATH untuk pipeline decoder lanjutan

Build EXE Windows:

```powershell
cd phone-mirror
.\scripts\build-desktop.ps1
```

Output:

```text
desktop/build/bin/voiXPe3per.exe
```

Install agar bisa dijalankan global:

```powershell
.\scripts\install-global.ps1
```

Setelah itu buka terminal baru dan jalankan:

```powershell
voiXPe3per.exe
```

Alias pendek juga tersedia:

```powershell
voixpe3per
```

Saat aplikasi dibuka, backend Wails akan:

- membuat room pairing relay,
- membuat QR payload `{ mode, relay, room }`,
- connect ke relay WebSocket publik,
- menyimpan trusted device di config lokal user.

Kalau muncul `Koneksi pairing terputus`, artinya relay WSS publik di QR belum hidup atau tunnel sudah mati. Cara paling cepat tanpa deploy akun hosting adalah menjalankan direct public WSS tunnel:

```powershell
cd phone-mirror
powershell -ExecutionPolicy Bypass -File .\scripts\start-public-wss.ps1
```

Script ini akan:

- membuat URL `wss://...trycloudflare.com/ws`,
- membuka EXE dengan `VOIXPE3PER_PAIRING_MODE=direct`,
- membuat QR baru yang masuk ke desktop lewat WSS publik,
- tidak membutuhkan APK, akun, atau jaringan yang sama.

Untuk dipakai lintas jaringan oleh semua orang, deploy relay dari folder `relay/`, lalu set environment variable desktop:

```powershell
$env:VOIXPE3PER_PAIRING_MODE="relay"
$env:VOIXPE3PER_RELAY_URL="wss://voixpe3per-relay.onrender.com/ws"
$env:VOIXPE3PER_PAIRING_PAGE_URL="https://voixxpe3per.vercel.app/pair"
voiXPe3per.exe
```

Jika `VOIXPE3PER_RELAY_URL` tidak diisi, desktop production otomatis memakai:

```text
wss://voixpe3per-relay.onrender.com/ws
```

QR Code sengaja dibuat sebagai URL halaman pairing, bukan JSON mentah. Ini membuat iOS Camera membuka halaman Vercel, lalu pairing bisa dilakukan langsung dari browser dengan tombol `Izinkan pairing`.

## Web pairing tanpa install aplikasi

Flow paling sederhana:

1. Buka EXE desktop.
2. Scan QR memakai kamera iOS/Android.
3. Browser masuk ke halaman Vercel `/pair`.
4. Tekan `Izinkan pairing`.
5. Desktop menyimpan browser/mobile sebagai trusted device.

Halaman Vercel hanya bisa pairing lewat relay `wss://...` publik. Vercel dipakai sebagai UI pairing statis; WebSocket relay tetap harus berjalan di host yang mendukung koneksi WebSocket persisten.

Untuk mode direct tunnel, halaman Vercel connect langsung ke desktop melalui `VOIXPE3PER_PUBLIC_WS_URL` dan token QR. Ini cocok untuk testing cepat saat relay publik permanen belum live.

## Public WSS Relay

Relay publik disiapkan untuk Render melalui `render.yaml`. Deploy sebagai Render Blueprint dari repo GitHub ini, lalu domain default akan menjadi:

```text
https://voixpe3per-relay.onrender.com
```

WebSocket URL:

```text
wss://voixpe3per-relay.onrender.com/ws
```

Jalur pairing publik membutuhkan URL `wss://`. Browser pairing Vercel tidak memakai `ws://` lokal.

## Android

Prerequisite:

- Android Studio atau Gradle dengan Android plugin
- Android SDK 35
- Device Android 8.0+ untuk MediaProjection foreground service

Build APK debug:

```bash
cd phone-mirror
./scripts/build-android.sh
```

Di Windows:

```powershell
cd phone-mirror
powershell -ExecutionPolicy Bypass -File .\scripts\build-android.ps1
```

Android flow:

1. Jalankan desktop dengan `scripts\start-public-wss.ps1`.
2. Buka app Android voiXPe3per.
3. Tap `Scan QR Pairing` dan scan QR dari desktop.
4. Pairing ke WSS publik dan simpan `trustSecret`.
5. Minta permission MediaProjection.
6. Encode layar ke H264 low latency.
7. Kirim frame binary ke desktop sampai layar muncul.

Reconnect trusted device memakai `device.reconnect`, jadi scan QR hanya diperlukan untuk pairing pertama.

## iOS Native Optional

Prerequisite:

- macOS dengan Xcode 15+
- XcodeGen (`brew install xcodegen`)
- iPhone/iPad iOS 16+

Generate project dan build:

```bash
cd phone-mirror/ios
xcodegen generate
open voiXPe3per.xcodeproj
```

Atau:

```bash
cd phone-mirror
./scripts/build-ios.sh
```

iOS native flow:

1. Scan QR dari desktop memakai app iOS voiXPe3per.
2. Connect ke relay WSS publik.
3. Join room relay sebagai role `ios`.
4. Kirim `pair.verify` dengan identity iOS.
5. Simpan `trustSecret` di Keychain.
6. Reconnect memakai `device.reconnect` tanpa scan ulang.

Catatan: pairing iOS sudah tersedia. Live screen mirroring iOS membutuhkan ReplayKit Broadcast Upload Extension karena iOS tidak mengizinkan app biasa menangkap layar penuh secara langsung seperti Android MediaProjection.

## Catatan V1

Frontend sudah memiliki WebCodecs renderer untuk H264 dan fallback visual bila browser runtime belum bisa decode payload mentah. Untuk produksi, langkah berikutnya adalah menambahkan transmux/Annex-B normalization atau pipeline FFmpeg yang lebih ketat untuk semua varian device encoder.
