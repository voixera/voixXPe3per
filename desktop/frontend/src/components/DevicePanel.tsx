import { Apple, Globe2, Smartphone, Trash2 } from "lucide-react";
import { useAppState } from "../store/appStore";
import type { TrustedDevice } from "../types";
import { StatusPill } from "./StatusPill";

export function DevicePanel({ devices }: { devices: TrustedDevice[] }) {
  const { actions } = useAppState();

  return (
    <aside className="flex h-full flex-col bg-shell-850">
      <div className="border-b border-black/40 px-4 py-3">
        <p className="text-[11px] uppercase tracking-[0.18em] text-slate-500">Devices</p>
      </div>

      <div className="flex-1 overflow-y-auto p-2">
        {devices.length === 0 ? (
          <div className="px-2 py-5 text-sm text-slate-500">No trusted devices</div>
        ) : (
          devices.map((device) => (
            <div
              key={device.id}
              className="group mb-1 grid grid-cols-[32px_1fr_28px] items-center gap-2 border border-transparent px-2 py-2 transition hover:border-shell-600 hover:bg-shell-800"
            >
              <div className="grid h-8 w-8 place-items-center bg-shell-900 text-slate-400">
                {device.platform === "web" ? (
                  <Globe2 size={16} />
                ) : device.platform === "ios" ? (
                  <Apple size={16} />
                ) : (
                  <Smartphone size={16} />
                )}
              </div>
              <div className="min-w-0">
                <p className="truncate text-sm font-medium text-slate-200">{device.name}</p>
                <p className="truncate text-[11px] text-slate-500">
                  {device.osName || platformLabel(device.platform)} {device.osVersion || device.androidVersion}
                </p>
                <div className="mt-1">
                  <StatusPill status={device.status} />
                </div>
              </div>
              <button
                className="grid h-7 w-7 place-items-center text-slate-600 opacity-0 transition hover:bg-shell-700 hover:text-signal-red group-hover:opacity-100"
                title="Forget device"
                type="button"
                onClick={() => void actions.forgetDevice(device.id)}
              >
                <Trash2 size={14} />
              </button>
            </div>
          ))
        )}
      </div>

      <div className="border-t border-black/40 px-4 py-3 text-xs text-slate-500">
        Local-first pairing. No account, no cloud backend.
      </div>
    </aside>
  );
}

function platformLabel(platform: string) {
  if (platform === "ios") {
    return "iOS";
  }
  if (platform === "web") {
    return "Web";
  }
  if (platform === "android") {
    return "Android";
  }
  return "Mobile";
}
