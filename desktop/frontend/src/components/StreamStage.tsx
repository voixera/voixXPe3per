import { MonitorUp, RotateCw } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { H264Renderer } from "../services/h264Renderer";
import { useAppState } from "../store/appStore";
import type { StreamFrame, StreamMetrics, TrustedDevice } from "../types";
import { StatusPill } from "./StatusPill";

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
      <div className="flex items-center justify-between border-b border-black/40 bg-shell-850 px-5 py-3">
        <div className="min-w-0">
          <div className="flex items-center gap-3">
            <h2 className="truncate text-sm font-semibold text-slate-100">{device.name}</h2>
            <StatusPill status="connected" />
          </div>
          <p className="mt-1 text-xs text-slate-500">
            {device.manufacturer} {device.model} | Android {device.androidVersion}
          </p>
        </div>
        <div className="flex gap-2">
          <button className="toolbar-button" type="button" onClick={() => void actions.refreshStream()}>
            <RotateCw size={15} />
            Refresh
          </button>
          <button className="toolbar-button" type="button" onClick={() => void actions.toggleFullscreen()}>
            <MonitorUp size={15} />
            Fullscreen
          </button>
        </div>
      </div>

      <div className="relative flex-1 bg-[#08090b]">
        {!frame && (
          <div className="absolute inset-0 z-10 grid place-items-center bg-black/25 text-center">
            <div className="stream-loader">
              <div className="mx-auto mb-4 h-10 w-10 animate-spin border-2 border-shell-600 border-t-signal-green" />
              <p className="font-mono text-xs uppercase tracking-[0.2em] text-slate-400">Waiting for H264 frames</p>
            </div>
          </div>
        )}
        <canvas ref={canvasRef} className="h-full w-full" />
      </div>

      <footer className="flex h-[42px] items-center justify-between border-t border-black/50 bg-shell-850 px-5 font-mono text-xs text-slate-400">
        <span>FPS {metrics.fps}</span>
        <span>{metrics.codec || "H264"}</span>
        <span>{metrics.transport || "WiFi Local"}</span>
        <span>{metrics.latencyMs}ms</span>
        <span>{metrics.frames} frames</span>
      </footer>
    </div>
  );
}
