# Android App

The Android app is the screen-streaming client. Open it on the phone, scan the QR from the desktop EXE, approve pairing, then approve Android screen capture. The app sends H264 frames to the same public WSS transport used by the desktop.

Build on Windows:

```powershell
cd phone-mirror
powershell -ExecutionPolicy Bypass -File .\scripts\build-android.ps1
```

Install to a connected Android device when `adb` is available:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\install-android.ps1
```

The Android project uses the standard Gradle application layout:

```text
app/src/main/java/com/voixpe3per/
  pairing/
  capture/
  encoder/
  network/
  security/
```

This maps to the requested Android module structure while keeping Android Studio import/build behavior conventional.
