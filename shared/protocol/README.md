# voiXPe3per Local Protocol

voiXPe3per supports two transports. The default is global relay mode, so Android/iOS and desktop do not need to be on the same WiFi.

Global relay endpoint:

```text
wss://voixpe3per-relay.onrender.com/ws
```

The relay is a dumb room forwarder. It does not know device trust secrets and only forwards packets between peers in the same room.

Legacy LAN endpoint:

```text
ws://<desktop-lan-ip>:8080/ws
```

The relay QR payload:

```json
{
  "mode": "relay",
  "relay": "wss://voixpe3per-relay.onrender.com/ws",
  "room": "A1B2C3D4E5F6"
}
```

The LAN QR payload:

```json
{
  "mode": "lan",
  "host": "192.168.x.x",
  "port": 8080,
  "token": "generated_token"
}
```

## Pairing

Android, iOS, or the Vercel web pairing page sends `pair.verify` after scanning the QR. In relay mode, the relay room replaces the local LAN token:

```json
{
  "type": "pair.verify",
  "token": "generated_token_or_empty_for_relay",
  "room": "A1B2C3D4E5F6",
  "device": {
    "id": "android-device-uuid",
    "name": "Samsung Galaxy A55",
    "model": "Galaxy A55",
    "manufacturer": "Samsung",
    "platform": "android",
    "osName": "Android",
    "osVersion": "14",
    "androidVersion": "14"
  },
  "capabilities": {
    "encoder": "h264",
    "maxFps": 60
  }
}
```

Desktop replies with `pair.success` and returns a `trustSecret`. Android stores it in private app storage, iOS stores it in Keychain, and desktop stores only a SHA-256 hash in the user config folder.

The Vercel web pairing page stores its browser identity and `trustSecret` in `localStorage`. This keeps the "scan QR, click Izinkan pairing" flow app-free for iOS Camera and mobile browsers.

iOS uses the same message shape with `"platform": "ios"`:

```json
{
  "type": "pair.verify",
  "token": "",
  "room": "A1B2C3D4E5F6",
  "device": {
    "id": "ios-device-uuid",
    "name": "Faisal's iPhone",
    "model": "iPhone",
    "manufacturer": "Apple",
    "platform": "ios",
    "osName": "iOS",
    "osVersion": "18.5"
  },
  "capabilities": {
    "encoder": "h264",
    "maxFps": 60
  }
}
```

## Reconnect

Android and iOS reconnect without a QR scan:

```json
{
  "type": "device.reconnect",
  "deviceId": "android-device-uuid",
  "trustSecret": "local-secret-from-pairing"
}
```

Desktop replies with `reconnect.success` when the stored hash matches.

## Streaming

Before binary frames, Android sends:

```json
{
  "type": "stream.start",
  "codec": "H264",
  "width": 1080,
  "height": 2400,
  "targetFps": 60
}
```

H264 frames are binary packets:

```text
byte 0      'V'
byte 1      'X'
byte 2      protocol version, currently 1
byte 3      flags, bit 0 = key frame
byte 4-11   big-endian timestamp ns
byte 12..   encoded H264 payload
```
