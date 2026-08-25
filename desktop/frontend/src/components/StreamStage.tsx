import { Activity, MonitorUp, RotateCw } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { H264Renderer, type RendererStats } from "../services/h264Renderer";
import { useAppState } from "../store/appStore";
import type { StreamFrame, StreamMetrics, TrustedDevice } from "../types";

function PaneBadge({ text, warn }: { text: string; warn?: boolean }) {
  return (
    <span
      className={`absolute left-4 top-4 z-10 flex items-center gap-2 border px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.22em] ${
        warn ? "border-amber/60 bg-ink-950/90 text-amber" : "border-line-hi bg-ink-950/85 text-acid"
      }`}
    >
      <span className={`led ${warn ? "off" : "on"}`} /> {text}
    </span>
  );
}

function Spinner({ line1, line2, hint }: { line1: string; line2?: string; hint?: string }) {
  return (
    <div className="absolute inset-0 z-10 grid place-items-center text-center">
      <div>
        <div className="mx-auto mb-5 h-10 w-10 animate-spin border border-line-mid border-t-acid" />
        <p className="label-tech">
          {line1}
          <span className="cursor-blink" />
        </p>
        {line2 && (
          <p className="mx-auto mt-3 max-w-xs font-mono text-[10px] uppercase leading-relaxed tracking-[0.16em] text-dim/70">
            {line2}
          </p>
        )}
        {hint && (
          <p className="mx-auto mt-1 max-w-xs font-mono text-[10px] uppercase leading-relaxed tracking-[0.16em] text-dim/50">
            {hint}
          </p>
        )}
      </div>
    </div>
  );
}

const STREAM_STATE_LABEL: Record<string, string> = {
  idle: "IDLE",
  connected: "CONNECTED",
  starting: "STARTING",
  streaming: "STREAMING"
};

export function StreamStage({
  device,
  frame,
  metrics,
  streamState
}: {
  device: TrustedDevice;
  frame: StreamFrame | null;
  metrics: StreamMetrics;
  streamState: { state: string; activeDevice: string; lastFrameAgeMs: number };
}) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const renderer = useMemo(() => new H264Renderer(), []);
  const { state, actions } = useAppState();

  const camFrame = state.camFrame;
  const camExpected = state.camActive || !!camFrame;
  const showBoth = !!frame && !!camFrame;
  const [debugOpen, setDebugOpen] = useState(false);
  const [stats, setStats] = useState<RendererStats>(renderer.stats);

  useEffect(() => {
    renderer.onStats = setStats;
    return () => {
      renderer.onStats = null;
    };
  }, [renderer]);

  // Camera stall watchdog (phone locked / browser suspended).
  const [stalled, setStalled] = useState(false);
  const lastCamAt = useRef(0);
  useEffect(() => {
    if (camFrame) {
      lastCamAt.current = Date.now();
      setStalled(false);
    }
  }, [camFrame]);
  useEffect(() => {
    const t = setInterval(() => {
      setStalled(camExpected && lastCamAt.current > 0 && Date.now() - lastCamAt.current > 4000);
    }, 1500);
    return () => clearInterval(t);
  }, [camExpected]);

  // The screen pane owns its canvas unconditionally — camera activity must
  // never leave it detached (that froze rendering when cam started first).
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) {
      return;
    }
    const resize = () => {
      const rect = canvas.getBoundingClientRect();
      canvas.width = Math.max(320, Math.floor(rect.width * window.devicePixelRatio));
      canvas.height = Math.max(240, Math.floor(rect.height * window.devicePixelRatio));
      renderer.attach(canvas);
    };
    resize();
    window.addEventListener("resize", resize);
    return () => window.removeEventListener("resize", resize);
  }, [renderer]);

  useEffect(() => {
    if (frame) {
      renderer.push(frame);
    }
  }, [frame, renderer]);

  const streaming = streamState.state === "streaming";
  const screenStatus =
    streamState.state === "streaming"
      ? "STREAMING"
      : streamState.state === "starting"
        ? "STARTING — waiting for first frame"
        : device.status === "connected"
          ? "CONNECTED — waiting for screen stream"
          : "DEVICE OFFLINE";

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-line-mid bg-ink-900 px-5 py-2.5">
        <p className="label-tech">03 / Feed — {device.name}</p>
        <div className="flex gap-2">
          <button
            className={`btn-hard h-7 px-3 ${debugOpen ? "border-acid text-acid" : ""}`}
            type="button"
            onClick={() => setDebugOpen((open) => !open)}
          >
            <Activity size={13} />
            Debug
          </button>
          {device.streamCapable && (
            <>
              <button className="btn-hard h-7 px-3" type="button" onClick={() => void actions.refreshStream()}>
                <RotateCw size={13} />
                Keyframe
              </button>
              <button className="btn-hard h-7 px-3" type="button" onClick={() => void actions.toggleFullscreen()}>
                <MonitorUp size={13} />
                Fullscreen
              </button>
            </>
          )}
        </div>
      </div>

      <div className={`relative min-h-0 flex-1 ${showBoth ? "grid grid-cols-2 divide-x divide-line-mid" : ""}`}>
        {/* Screen recording pane (Android APK / iOS broadcast / H264) */}
        <section className="relative bg-ink-950">
          <canvas ref={canvasRef} className="h-full w-full" />
          {!frame && (
            <Spinner
              line1={
                device.status !== "connected"
                  ? "No trusted device online"
                  : streamState.state === "streaming"
                    ? "Receiving frames…"
                    : "Connected — awaiting screen frames"
              }
              line2={
                device.platform === "ios"
                  ? "iOS: start Screen Broadcast in Control Center"
                  : "Start capture in the phone app"
              }
              hint={`state: ${screenStatus}`}
            />
          )}
          {frame && <PaneBadge text="Screen" warn={!streaming} />}
          {debugOpen && (
            <div className="absolute bottom-4 left-4 z-10 border border-line-hi bg-ink-950/95 px-3 py-2 font-mono text-[10px] leading-relaxed tracking-[0.14em] text-dim">
              <p className="text-bone">SCREEN DEBUG</p>
              <p>
                STATE: <b className={streaming ? "text-acid" : "text-amber"}>{STREAM_STATE_LABEL[streamState.state] ?? streamState.state.toUpperCase()}</b>
              </p>
              <p>DEV: {device.name} ({device.platform})</p>
              <p>
                FPS <b className="text-acid">{metrics.fps}</b> · FRAMES <b className="text-acid">{metrics.frames}</b> · LAT {metrics.latencyMs}ms
              </p>
              <p>
                RES {stats.width > 0 ? `${stats.width}x${stats.height}` : metrics.resolution || "?"} · FMT {metrics.codec || "H264"}
              </p>
              <p>
                LAST FRAME {stats.lastFrameAt > 0 ? `${Date.now() - stats.lastFrameAt}ms ago` : "never"} · AGE {streamState.lastFrameAgeMs}ms
              </p>
              <p>
                DECODED <b className="text-acid">{stats.decoded}</b> · ERR {stats.decodeErrors} · DROP {stats.dropped}
              </p>
            </div>
          )}
        </section>

        {/* Camera pane (phone browser / JPEG over Supabase) */}
        {camExpected && (
          <section className="relative bg-ink-950">
            <img
              ref={(el) => {
                if (el && camFrame) {
                  el.src = `data:image/jpeg;base64,${camFrame.j}`;
                }
              }}
              alt=""
              className="absolute inset-0 h-full w-full object-contain"
              style={{ display: camFrame ? "block" : "none" }}
            />
            {!camFrame && (
              <Spinner line1="Awaiting phone camera" line2={state.camFrames > 0 ? undefined : "Keep the pairing page open on the phone"} />
            )}
            {!camFrame && state.camStatus && (
              <p className={`mt-2 font-mono text-[10px] ${state.camStatus === "SUBSCRIBED" ? "text-dim/70" : "text-amber"}`}>
                RT: {state.camStatus} · FRM {state.camFrames}
              </p>
            )}
            {camFrame && <PaneBadge text="Camera" />}
            {stalled && (
              <span className="absolute bottom-4 left-4 z-10 flex items-center gap-2 border border-amber/60 bg-ink-950/90 px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.2em] text-amber">
                <span className="led off" /> Stalled — unlock phone
              </span>
            )}
          </section>
        )}
      </div>

      <footer className="flex h-9 items-center justify-between border-t border-line-mid bg-ink-900 px-5 font-mono text-[11px] uppercase tracking-[0.18em] text-dim">
        <span>FPS <b className="text-acid">{metrics.fps}</b></span>
        <span>{metrics.codec || "H264"}</span>
        {camFrame ? (
          <>
            <span>CAM {camFrame.w > 0 ? <b className="text-acid">{camFrame.w}x{camFrame.h}</b> : ""}</span>
            <span>FRM <b className="text-acid">{state.camFrames}</b></span>
          </>
        ) : (
          <>
            <span>{stats.width > 0 ? `${stats.width}x${stats.height}` : metrics.resolution || "Auto"}</span>
            <span>{metrics.latencyMs}ms</span>
          </>
        )}
        <span className={`flex items-center gap-1.5 ${streaming ? "text-alarm" : "text-dim"}`}>
          <span className={`led ${streaming ? "on" : "off"}`} style={{ background: streaming ? "#ff5d45" : undefined, boxShadow: streaming ? "0 0 8px rgba(255,93,69,.7)" : undefined }} />
          REC
        </span>
      </footer>
    </div>
  );
}
