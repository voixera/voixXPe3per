# voiXPe3per iOS

Starter iOS client untuk pairing ke desktop voiXPe3per.

## Fitur awal

- Scan QR pairing dengan AVFoundation.
- Baca QR berbentuk URL `https://.../pair?...`, custom scheme `voixpe3per://pair?...`, atau JSON legacy.
- Connect ke WebSocket relay/LAN.
- Kirim `relay.join` dengan role `ios`.
- Kirim `pair.verify` dengan identity iOS.
- Simpan trusted desktop untuk reconnect tanpa scan ulang.

## Build di macOS

Project ini memakai XcodeGen agar file project tidak perlu ditulis manual.

```bash
cd phone-mirror/ios
brew install xcodegen
xcodegen generate
open voiXPe3per.xcodeproj
```

Atau build dari terminal:

```bash
xcodegen generate
xcodebuild -scheme voiXPe3per -destination 'platform=iOS Simulator,name=iPhone 15' build
```

## Catatan streaming iOS

iOS tidak mengizinkan app biasa mengambil layar penuh secara diam-diam seperti Android MediaProjection. Untuk live screen mirroring iOS, tahap berikutnya adalah menambahkan ReplayKit Broadcast Upload Extension yang mengirim sample buffer H264 ke WebSocket yang sama setelah pairing berhasil.
