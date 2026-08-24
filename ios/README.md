# voiXPe3per iOS

Client iOS native lengkap dengan **Screen Broadcast Extension (ReplayKit)** & **H264 Hardware Encoder (VideoToolbox)** untuk mirror layar penuh iPhone/iPad ke desktop Windows secara realtime.

## Arsitektur iOS

1. **Main App (`voiXPe3per`)**:
   - UI SwiftUI + Scanner QR kamera.
   - Pairing & Reconnect ke Desktop via WSS / Relay.
   - Menyimpan `trustSecret` di Keychain.
   - Menyediakan tombol `RPSystemBroadcastPickerView` untuk mulai screen broadcast.

2. **Broadcast Extension (`voiXPe3perBroadcast`)**:
   - `RPBroadcastSampleHandler` menangkap full frame layar OS.
   - `H264Encoder` meng-encode video stream memakai Apple Hardware VideoToolbox.
   - `DesktopSocket` & `FramePacket` mengirim binary stream H264 langsung ke desktop.

## Cara Build & Install ke iPhone/iPad (macOS)

Diperlukan Mac dengan Xcode 15+:

```bash
cd phone-mirror/ios

# Install generator jika belum ada
brew install xcodegen

# Generate Xcode project (App + Extension)
xcodegen generate

# Buka di Xcode
open voiXPe3per.xcodeproj
```

Di Xcode:
1. Pilih Development Team Anda di tab **Signing & Capabilities** untuk target `voiXPe3per` dan `voiXPe3perBroadcast`.
2. Pasang iPhone via kabel USB / WiFi.
3. Klik **Run** (Cmd + R) ke device fisik.

## Cara Pakai

1. Buka aplikasi `voiXPe3per` di iPhone.
2. Tap **Scan QR Desktop** lalu scan QR dari layar desktop Windows.
3. Setelah status `Paired with desktop`, tap ikon **Mulai Screen Broadcast**.
4. Pilih `voiXPe3per Broadcast` lalu tekan **Start Broadcast**.
5. Layar iPhone muncul dan streaming langsung ke desktop Windows.
