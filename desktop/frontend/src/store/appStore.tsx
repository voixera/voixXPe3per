import { createContext, useContext, useEffect, useMemo, useReducer, type ReactNode } from "react";
import { desktopApi } from "../services/desktopApi";
import { subscribeCam } from "../services/supabaseCam";
import type { AuthIdentity, CamFrame, DesktopSnapshot, StreamFrame, StreamMetrics } from "../types";

type AppState = DesktopSnapshot & {
  latestFrame: StreamFrame | null;
  camFrame: CamFrame | null;
  camFrames: number;
  camStatus: string;
  booting: boolean;
};

type Action =
  | { type: "snapshot"; snapshot: DesktopSnapshot }
  | { type: "metrics"; metrics: StreamMetrics }
  | { type: "frame"; frame: StreamFrame }
  | { type: "cam-frame"; frame: CamFrame }
  | { type: "cam-status"; status: string }
  | { type: "booted" };

const initialState: AppState = {
  pairing: {
    host: "",
    port: 8080,
    token: "",
    mode: "relay",
    relayUrl: "",
    room: "",
    qrDataUrl: "",
    status: "Standby"
  },
  devices: [],
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
  camActive: false,
  camFrame: null,
  camFrames: 0,
  camStatus: "",
  latestFrame: null,
  booting: true
};

const AppStateContext = createContext<
  | {
      state: AppState;
      actions: {
        refreshPairing(): Promise<void>;
        startFreshPairing(): Promise<void>;
        refreshStream(): Promise<void>;
        toggleFullscreen(): Promise<void>;
        forgetDevice(deviceId: string): Promise<void>;
        loginWithDiscord(): Promise<void>;
        logout(): Promise<void>;
      };
    }
  | undefined
>(undefined);

export function AppStateProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(reducer, initialState);

  useEffect(() => {
    let disposed = false;
    desktopApi.getSnapshot().then((snapshot) => {
      if (!disposed) {
        dispatch({ type: "snapshot", snapshot });
        dispatch({ type: "booted" });
      }
    });

    const disposeSnapshot = desktopApi.onSnapshot((snapshot) => dispatch({ type: "snapshot", snapshot }));
    const disposeMetrics = desktopApi.onMetrics((metrics) => dispatch({ type: "metrics", metrics }));
    const disposeFrame = desktopApi.onFrame((frame) => dispatch({ type: "frame", frame }));

    return () => {
      disposed = true;
      disposeSnapshot();
      disposeMetrics();
      disposeFrame();
    };
  }, []);

  // Camera frames arrive via supabase-js inside this webview (browser
  // websocket passes Cloudflare; Go dials do not).
  const { auth, pairing } = state;
  useEffect(() => {
    if (!auth.cloudReady || !auth.supabaseUrl || !auth.anonKey || !pairing.room || pairing.mode !== "cloud") {
      return;
    }
    return subscribeCam(auth.supabaseUrl, auth.anonKey, pairing.room, (frame) => {
      dispatch({ type: "cam-frame", frame });
    }, (status) => {
      dispatch({ type: "cam-status", status });
    });
  }, [auth.cloudReady, auth.supabaseUrl, auth.anonKey, pairing.mode, pairing.room]);

  const actions = useMemo(
    () => ({
      async loginWithDiscord() {
        await desktopApi.loginWithDiscord();
      },
      async logout() {
        await desktopApi.logout();
        const snapshot = await desktopApi.getSnapshot();
        dispatch({ type: "snapshot", snapshot });
      },
      async refreshPairing() {
        const pairing = await desktopApi.refreshPairing();
        dispatch({
          type: "snapshot",
          snapshot: {
            pairing,
            devices: state.devices,
            metrics: state.metrics,
            auth: state.auth,
            camActive: state.camActive
          }
        });
      },
      async startFreshPairing() {
        const pairing = await desktopApi.startFreshPairing();
        dispatch({
          type: "snapshot",
          snapshot: {
            pairing,
            devices: state.devices,
            metrics: state.metrics,
            auth: state.auth,
            camActive: state.camActive
          }
        });
      },
      async refreshStream() {
        await desktopApi.refreshStream();
      },
      async toggleFullscreen() {
        await desktopApi.toggleFullscreen();
      },
      async forgetDevice(deviceId: string) {
        await desktopApi.forgetDevice(deviceId);
        const snapshot = await desktopApi.getSnapshot();
        dispatch({ type: "snapshot", snapshot });
      }
    }),
    [state.devices, state.metrics, state.auth]
  );

  return <AppStateContext.Provider value={{ state, actions }}>{children}</AppStateContext.Provider>;
}

export function useAppState() {
  const value = useContext(AppStateContext);
  if (!value) {
    throw new Error("useAppState must be used inside AppStateProvider");
  }
  return value;
}

function reducer(state: AppState, action: Action): AppState {
  switch (action.type) {
    case "snapshot": {
      const roomChanged = action.snapshot.pairing.room !== state.pairing.room;
      const next: AppState = {
        ...state,
        ...action.snapshot,
        auth: action.snapshot.auth ?? state.auth,
        booting: false
      };
      if (roomChanged) {
        next.camFrame = null;
        next.camFrames = 0;
        next.camStatus = "";
      }
      if (!action.snapshot.camActive && !state.camActive) {
        next.camFrame = roomChanged ? null : state.camFrame;
      }
      return next;
    }
    case "metrics":
      return { ...state, metrics: action.metrics };
    case "frame":
      return { ...state, latestFrame: action.frame };
    case "cam-frame":
      return { ...state, camFrame: action.frame, camFrames: state.camFrames + 1 };
    case "cam-status":
      return { ...state, camStatus: action.status };
    case "booted":
      return { ...state, booting: false };
    default:
      return state;
  }
}
