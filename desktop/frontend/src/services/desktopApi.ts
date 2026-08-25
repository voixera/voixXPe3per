import type { DesktopSnapshot, PairingSession, StreamFrame, StreamMetrics } from "../types";

const emptySnapshot: DesktopSnapshot = {
  pairing: {
    host: "",
    port: 8080,
    token: "dev-token",
    mode: "relay",
    relayUrl: "wss://voixpe3per-relay.onrender.com/ws",
    room: "DEVROOM",
    qrDataUrl: "",
    status: "Waiting for device..."
  },
  devices: [
    {
      id: "demo-galaxy-a55",
      name: "Samsung Galaxy A55",
      model: "Galaxy A55",
      manufacturer: "Samsung",
      platform: "android",
      osName: "Android",
      osVersion: "14",
      androidVersion: "14",
      streamCapable: true,
      cameraOk: true,
      micOk: true,
      status: "offline",
      lastSeen: new Date().toISOString()
    }
  ],
  metrics: {
    fps: 0,
    codec: "H264",
    transport: "Public WSS",
    latencyMs: 0,
    frames: 0,
    updatedAt: new Date().toISOString(),
    resolution: "Auto"
  },
  auth: {
    loggedIn: false,
    email: "",
    name: "",
    avatar: "",
    providerId: "",
    cloudReady: true,
    supabaseUrl: "",
    anonKey: ""
  },
  camActive: false
};

export const desktopApi = {
  async getSnapshot(): Promise<DesktopSnapshot> {
    return window.go?.mainapp?.App?.GetSnapshot?.() ?? emptySnapshot;
  },

  async refreshPairing(): Promise<PairingSession> {
    return window.go?.mainapp?.App?.RefreshPairing?.() ?? emptySnapshot.pairing;
  },

  async startFreshPairing(): Promise<PairingSession> {
    return window.go?.mainapp?.App?.StartFreshPairing?.() ?? emptySnapshot.pairing;
  },

  async forgetDevice(deviceId: string): Promise<void> {
    await window.go?.mainapp?.App?.ForgetDevice?.(deviceId);
  },

  async refreshStream(): Promise<void> {
    await window.go?.mainapp?.App?.RefreshStream?.();
  },

  async toggleFullscreen(): Promise<void> {
    await window.go?.mainapp?.App?.ToggleFullscreen?.();
  },

  async loginWithDiscord(): Promise<void> {
    await window.go?.mainapp?.App?.LoginWithDiscord?.();
  },

  async logout(): Promise<void> {
    await window.go?.mainapp?.App?.Logout?.();
  },

  onSnapshot(callback: (snapshot: DesktopSnapshot) => void): () => void {
    return window.runtime?.EventsOn?.("app.snapshot", callback) ?? (() => undefined);
  },

  onFrame(callback: (frame: StreamFrame) => void): () => void {
    return window.runtime?.EventsOn?.("stream.frame", callback) ?? (() => undefined);
  },

  onMetrics(callback: (metrics: StreamMetrics) => void): () => void {
    return window.runtime?.EventsOn?.("stream.metrics", callback) ?? (() => undefined);
  }
};
