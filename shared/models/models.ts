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
  streamCapable: boolean;
}

export interface PairingPayload {
  mode: "relay" | "direct" | string;
  host?: string;
  port?: number;
  token?: string;
  relay?: string;
  public?: string;
  room?: string;
}

export interface TrustedDevice extends DeviceIdentity {
  status: DeviceStatus;
  lastSeen: string;
}

export interface StreamMetrics {
  fps: number;
  codec: "H264";
  transport: "Public WSS";
  latencyMs: number;
  frames: number;
  resolution: string;
}
