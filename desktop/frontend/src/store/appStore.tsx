import { createContext, useContext, useEffect, useMemo, useReducer, type ReactNode } from "react";
import { desktopApi } from "../services/desktopApi";
import type { DesktopSnapshot, StreamFrame, StreamMetrics } from "../types";

type AppState = DesktopSnapshot & {
  latestFrame: StreamFrame | null;
  booting: boolean;
};

type Action =
  | { type: "snapshot"; snapshot: DesktopSnapshot }
  | { type: "metrics"; metrics: StreamMetrics }
  | { type: "frame"; frame: StreamFrame }
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
    status: "Waiting for device..."
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
  latestFrame: null,
  booting: true
};

const AppStateContext = createContext<
  | {
      state: AppState;
      actions: {
        refreshPairing(): Promise<void>;
        refreshStream(): Promise<void>;
        toggleFullscreen(): Promise<void>;
        forgetDevice(deviceId: string): Promise<void>;
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

  const actions = useMemo(
    () => ({
      async refreshPairing() {
        const pairing = await desktopApi.refreshPairing();
        dispatch({
          type: "snapshot",
          snapshot: {
            pairing,
            devices: state.devices,
            metrics: state.metrics
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
    [state.devices, state.metrics]
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
    case "snapshot":
      return { ...state, ...action.snapshot, booting: false };
    case "metrics":
      return { ...state, metrics: action.metrics };
    case "frame":
      return { ...state, latestFrame: action.frame };
    case "booted":
      return { ...state, booting: false };
    default:
      return state;
  }
}
