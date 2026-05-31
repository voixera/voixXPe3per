import { Maximize2, RefreshCw, Smartphone } from "lucide-react";
import { useAppState } from "../store/appStore";
import type { StreamMetrics, TrustedDevice } from "../types";
import { StatusPill } from "./StatusPill";

export function TopBar({
  connectedDevice,
  metrics
}: {
  connectedDevice?: TrustedDevice;
  metrics: StreamMetrics;
}) {
  const { actions } = useAppState();

  return (
    <header className="flex h-[54px] items-center justify-between border-b border-black/50 bg-shell-850 px-4 shadow-insetPanel">
      <div className="flex items-center gap-3">
        <div className="grid h-8 w-8 place-items-center border border-shell-600 bg-shell-800">
          <Smartphone size={17} className="text-signal-green" />
        </div>
        <div>
          <h1 className="text-sm font-semibold leading-4 text-slate-100">voiXPe3per</h1>
          <p className="text-[11px] text-slate-500">Android and iOS mirror console</p>
        </div>
      </div>

      <div className="flex items-center gap-5 text-sm">
        <div className="hidden min-w-[190px] text-right md:block">
          <p className="truncate text-slate-200">{connectedDevice?.name ?? "No device connected"}</p>
          <p className="text-[11px] text-slate-500">
            {connectedDevice ? `${connectedDevice.osName || connectedDevice.platform} ` : ""}
            {metrics.resolution || "Auto"} stream
          </p>
        </div>
        <StatusPill status={connectedDevice ? "connected" : "offline"} />
        <button
          className="icon-button"
          title="Refresh stream"
          type="button"
          onClick={() => void actions.refreshStream()}
        >
          <RefreshCw size={16} />
        </button>
        <button
          className="icon-button"
          title="Fullscreen"
          type="button"
          onClick={() => void actions.toggleFullscreen()}
        >
          <Maximize2 size={16} />
        </button>
      </div>
    </header>
  );
}
