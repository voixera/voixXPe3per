import { MonitorUp, RotateCw } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { H264Renderer } from "../services/h264Renderer";
import { useAppState } from "../store/appStore";
import type { StreamFrame, StreamMetrics, TrustedDevice } from "../types";

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
  const renderer = useMemo(() => new H264Renderer(), []);
  const { state, actions } = useAppState();

  // Camera feed wins as soon as real frames arrive, regardless of platform.
  const camFrame = state.camFrame;

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
      setStalled(!!state.camActive && lastFrameAt.current > 0 && Date.now() - lastFrameAt.current > 4000);
    }, 1500);
    return () => clearInterval(t);
  }, [state.camActive]);

  useEffect(() => {
    if (camFrame || !device.streamCapable) {
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
  }, [renderer, camFrame, device.streamCapable]);

  useEffect(() => {
    if (!camFrame && frame) {
      renderer.push(frame);
    }
  }, [frame, camFrame, renderer]);

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-line-mid bg-ink-900 px-5 py-2.5">
        <p className="label-tech">03 / Feed — {device.name}</p>
        {!camFrame && device.streamCapable && (
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

      <div className="relative min-h-0 flex-1 bg-ink-950">
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
          <div className="absolute inset-0 z-10 grid place-items-center text-center">
            <div>
              <div className="mx-auto mb-5 h-10 w-10 animate-spin border border-line-mid border-t-acid" />
              {state.camActive ? (
                <>
                  <p className="label-tech">Awaiting phone camera<span className="cursor-blink" /></p>
                  <p className="mt-3 max-w-xs font-mono text-[10px] leading-relaxed uppercase tracking-[0.16em] text-dim/70">
                    Keep the pairing page open on the phone with the screen on
                  </p>
                  {state.camStatus && state.camStatus !== "SUBSCRIBED" && (
                    <p className="mt-3 font-mono text-[10px] text-amber">RT: {state.camStatus}</p>
                  )}
                </>
              ) : !device.streamCapable ? (
                <p className="label-tech">Link established — no stream from this device</p>
              ) : (
                <p className="label-tech">Awaiting H264 frames<span className="cursor-blink" /></p>
              )}
            </div>
          </div>
        )}

        {camFrame && (
          <span className="absolute left-4 top-4 z-10 flex items-center gap-2 border border-line-hi bg-ink-950/85 px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.22em] text-acid">
            <span className="led on" /> Live / Phone Cam
          </span>
        )}
        {stalled && (
          <span className="absolute left-4 bottom-4 z-10 flex items-center gap-2 border border-amber/60 bg-ink-950/90 px-2.5 py-1 font-mono text-[10px] uppercase tracking-[0.2em] text-amber">
            <span className="led off" /> Signal stalled — unlock phone / reopen page
          </span>
        )}
      </div>

      <footer className="flex h-9 items-center justify-between border-t border-line-mid bg-ink-900 px-5 font-mono text-[11px] uppercase tracking-[0.18em] text-dim">
        {camFrame ? (
          <>
            <span>CAM <b className="text-acid">{camFrame.w}x{camFrame.h}</b></span>
            <span>JPEG / Supabase RT</span>
            <span>FRM <b className="text-acid">{state.camFrames}</b></span>
          </>
        ) : (
          <>
            <span>FPS <b className="text-acid">{metrics.fps}</b></span>
            <span>{metrics.codec || "H264"}</span>
            <span>{metrics.resolution || "Auto"}</span>
            <span>{metrics.latencyMs}ms</span>
            <span>FRM <b className="text-acid">{metrics.frames}</b></span>
          </>
        )}
        <span className="text-alarm">REC</span>
      </footer>
    </div>
  );
}
