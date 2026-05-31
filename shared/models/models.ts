export type DeviceStatus = "connected" | "offline";

export interface DeviceIdentity {
  id: string;
  name: string;
  model: string;
  manufacturer: string;
  platform: "android" | "ios" | "web" | string;
  osName: string;
  osVersion: string;
  androidVersion: string;
}

export interface PairingPayload {
  host: string;
  port: number;
  token: string;
}

export interface TrustedDevice extends DeviceIdentity {
  status: DeviceStatus;
  lastSeen: string;
}

export interface StreamMetrics {
  fps: number;
  codec: "H264";
  transport: "WiFi Local";
  latencyMs: number;
  frames: number;
  resolution: string;
}
