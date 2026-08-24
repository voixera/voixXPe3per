import { MonitorUp, RotateCw } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
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
  const { actions } = useAppState();

  const isPairingOnly = !device.streamCapable;

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

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-line-mid bg-ink-900 px-5 py-2.5">
        <p className="label-tech">03 / Feed — {device.name}</p>
        {!isPairingOnly && (
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
        {isPairingOnly ? (
          <div className="absolute inset-0 z-10 grid place-items-center text-center">
            <div className="panel max-w-md px-8 py-8">
              <p className="label-tech text-acid">Link Established</p>
              <h3 className="mt-4 font-display text-xl font-semibold uppercase tracking-[0.12em] text-bone">
                Browser cannot push H264
              </h3>
              <p className="mt-3 font-mono text-xs leading-relaxed text-dim">
                Mobile browsers are pairing-only. Full screen capture needs the Android app (MediaProjection) or the
                iOS client (ReplayKit).
              </p>
            </div>
          </div>
        ) : !frame ? (
          <div className="absolute inset-0 z-10 grid place-items-center text-center">
            <div>
              <div className="mx-auto mb-5 h-10 w-10 animate-spin border border-line-mid border-t-acid" />
              <p className="label-tech">Awaiting H264 frames<span className="cursor-blink" /></p>
            </div>
          </div>
        ) : null}
        <canvas ref={canvasRef} className="h-full w-full" />
      </div>

      <footer className="flex h-9 items-center justify-between border-t border-line-mid bg-ink-900 px-5 font-mono text-[11px] uppercase tracking-[0.18em] text-dim">
        <span>FPS <b className="text-acid">{metrics.fps}</b></span>
        <span>{metrics.codec || "H264"}</span>
        <span>{metrics.resolution || "Auto"}</span>
        <span>{metrics.latencyMs}ms</span>
        <span>FRM <b className="text-acid">{metrics.frames}</b></span>
        <span className="text-alarm">REC</span>
      </footer>
    </div>
  );
}
