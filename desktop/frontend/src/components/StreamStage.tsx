import { MonitorUp, RotateCw } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { H264Renderer } from "../services/h264Renderer";
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

function Spinner({ line1, line2 }: { line1: string; line2?: string }) {
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
      </div>
    </div>
  );
}

export function StreamStage({
  device,
  frame,
  metrics
}: {
  device: TrustedDevice;
  frame: StreamFrame | null;
  metrics: StreamMetrics;
}) {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const imgRef = useRef<HTMLImageElement | null>(null);
  const renderer = useMemo(() => new H264Renderer(), []);
  const { state, actions } = useAppState();

  const camFrame = state.camFrame;
  const camExpected = state.camActive || !!camFrame;
  const showBoth = !!frame && !!camFrame;

  // Warn when the phone stops sending (screen locked / browser suspended).
  const [stalled, setStalled] = useState(false);
  const lastFrameAt = useRef(0);
  useEffect(() => {
    if (camFrame) {
      lastFrameAt.current = Date.now();
      setStalled(false);
    }
  }, [camFrame]);
  useEffect(() => {
    const t = setInterval(() => {
      setStalled(camExpected && lastFrameAt.current > 0 && Date.now() - lastFrameAt.current > 4000);
    }, 1500);
    return () => clearInterval(t);
  }, [camExpected]);

  useEffect(() => {
    if (camFrame) {
      return;
    }
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
  }, [renderer, camFrame]);

  useEffect(() => {
    if (frame) {
      renderer.push(frame);
    }
  }, [frame, renderer]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-line-mid bg-ink-900 px-5 py-2.5">
        <p className="label-tech">03 / Feed — {device.name}</p>
        {device.streamCapable && (
          <div className="flex gap-2">
            <button className="btn-hard h-7 px-3" type="button" onClick={() => void actions.refreshStream()}>
              <RotateCw size={13} />
              Keyframe
            </button>
            <button className="btn-hard h-7 px-3" type="button" onClick={() => void actions.toggleFullscreen()}>
              <MonitorUp size={13} />
              Fullscreen
            </button>
          </div>
        )}
      </div>

      <div className={`relative min-h-0 flex-1 ${showBoth ? "grid grid-cols-2 divide-x divide-line-mid" : ""}`}>
        {/* Screen recording pane (Android app / H264) */}
        <section className="relative bg-ink-950">
          <canvas ref={canvasRef} className="h-full w-full" style={{ display: frame ? "block" : "block" }} />
          {!frame && (
            <Spinner
              line1={device.streamCapable ? "Awaiting screen frames" : "No screen stream from this device"}
              line2={device.streamCapable ? "Start capture in the Android app" : undefined}
            />
          )}
          {frame && <PaneBadge text="Screen" />}
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
              <Spinner
                line1="Awaiting phone camera"
                line2={state.camFrames > 0 ? undefined : "Keep the pairing page open on the phone"}
              />
            )}
            {state.camFrames > 0 && !camFrame && null}
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
            <span>{metrics.resolution || "Auto"}</span>
            <span>{metrics.latencyMs}ms</span>
          </>
        )}
        <span className="text-alarm">REC</span>
      </footer>
    </div>
  );
}
