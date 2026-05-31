# voiXPe3per

voiXPe3per adalah starter project untuk mirroring layar Android ke desktop Windows. Mode default sekarang memakai relay WebSocket publik, jadi Android dan desktop tidak harus berada di jaringan WiFi/LAN yang sama. Desktop menampilkan QR pairing, Android scan QR, lalu H264 frames dikirim melalui relay.

## Struktur

```text
phone-mirror/
  desktop/   Wails v2 + Go backend + React/TypeScript/Tailwind UI
  android/   Kotlin Android app, ZXing QR scanner, MediaProjection, H264 encoder
  shared/    Protocol docs, JSON schema, shared model contracts
  scripts/   Build helpers
```

## Desktop

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

Untuk dipakai lintas jaringan oleh semua orang, deploy relay dari folder `relay/`, lalu set environment variable desktop:

```powershell
$env:VOIXPE3PER_PAIRING_MODE="relay"
$env:VOIXPE3PER_RELAY_URL="wss://domain-relay-kamu/ws"
$env:VOIXPE3PER_PAIRING_PAGE_URL="https://voixxpe3per.vercel.app/pair"
voiXPe3per.exe
```

QR Code sengaja dibuat sebagai URL halaman pairing, bukan JSON mentah. Ini membuat iOS Camera membuka halaman pairing, sementara aplikasi Android tetap bisa membaca parameter pairing dari URL yang sama.

Untuk development relay lokal:

```powershell
.\scripts\start-relay.ps1
$env:VOIXPE3PER_RELAY_URL="ws://127.0.0.1:8090/ws"
voiXPe3per.exe
```

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

Android flow:

1. Scan QR dari desktop.
2. Connect ke relay `wss://<domain>/ws`.
3. Join room dari QR dan kirim `pair.verify`.
4. Simpan `trustSecret` setelah `pair.success`.
5. Minta permission MediaProjection.
6. Encode layar ke H264 low latency.
7. Kirim frame binary ke desktop.

Reconnect trusted device memakai `device.reconnect`, jadi scan QR hanya diperlukan untuk pairing pertama.

## Catatan V1

Frontend sudah memiliki WebCodecs renderer untuk H264 dan fallback visual bila browser runtime belum bisa decode payload mentah. Untuk produksi, langkah berikutnya adalah menambahkan transmux/Annex-B normalization atau pipeline FFmpeg yang lebih ketat untuk semua varian device encoder.
