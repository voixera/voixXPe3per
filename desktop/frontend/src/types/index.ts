export type DeviceStatus = "connected" | "offline";

export interface PairingSession {
  host: string;
  port: number;
  token: string;
  mode: "relay" | "direct" | "cloud";
  relayUrl: string;
  room: string;
  qrDataUrl: string;
  status: string;
}

export interface AuthIdentity {
  loggedIn: boolean;
  email: string;
  name: string;
  avatar: string;
  providerId: string;
  cloudReady: boolean;
}

export interface TrustedDevice {
  id: string;
  name: string;
  model: string;
  manufacturer: string;
  platform: "android" | "ios" | "web" | string;
  osName: string;
  osVersion: string;
  androidVersion: string;
  streamCapable: boolean;
  cameraOk: boolean;
  micOk: boolean;
  status: DeviceStatus;
  lastSeen: string;
}

export interface StreamMetrics {
  fps: number;
  codec: string;
  transport: string;
  latencyMs: number;
  frames: number;
  updatedAt: string;
  resolution: string;
}

export interface StreamFrame {
  codec: string;
  data: string;
  keyFrame: boolean;
  timestampNs: number;
  receivedAtNs: number;
}

export interface DesktopSnapshot {
  pairing: PairingSession;
  devices: TrustedDevice[];
  metrics: StreamMetrics;
  auth: AuthIdentity;
}

export interface WailsDesktopApi {
  GetSnapshot(): Promise<DesktopSnapshot>;
  RefreshPairing(): Promise<PairingSession>;
  ForgetDevice(deviceId: string): Promise<void>;
  RefreshStream(): Promise<void>;
  ToggleFullscreen(): Promise<void> | void;
  LoginWithDiscord(): Promise<void>;
  Logout(): Promise<void>;
}

export interface WailsRuntime {
  EventsOn<T>(eventName: string, callback: (payload: T) => void): () => void;
}
