import { Apple, Globe2, Smartphone, Trash2 } from "lucide-react";
import { useAppState } from "../store/appStore";
import type { TrustedDevice } from "../types";
import { StatusPill } from "./StatusPill";

export function DevicePanel({ devices }: { devices: TrustedDevice[] }) {
  const { actions } = useAppState();

  return (
    <aside className="flex h-full flex-col bg-ink-900">
      <div className="border-b border-line-mid px-4 py-2.5">
        <p className="label-tech">02 / Devices</p>
      </div>

      <div className="flex-1 overflow-y-auto">
        {devices.length === 0 ? (
          <p className="px-4 py-6 font-mono text-xs text-dim">No trusted devices yet</p>
        ) : (
          devices.map((device) => (
            <div
              key={device.id}
              className="group flex items-start gap-3 border-b border-line-dim px-4 py-3 transition-colors hover:bg-ink-800"
            >
              <span className={`led mt-[7px] ${device.status === "connected" ? "on" : "off"}`} />
              <div className="min-w-0 flex-1">
                <div className="flex items-center gap-2">
                  {device.platform === "web" ? (
                    <Globe2 size={13} className="shrink-0 text-dim" />
                  ) : device.platform === "ios" ? (
                    <Apple size={13} className="shrink-0 text-dim" />
                  ) : (
                    <Smartphone size={13} className="shrink-0 text-dim" />
                  )}
                  <p className="truncate font-mono text-xs text-bone">{device.name}</p>
                </div>
                <p className="mt-0.5 truncate pl-[21px] font-mono text-[10px] uppercase tracking-wider text-dim/70">
                  {device.osName || platformLabel(device.platform)} {device.osVersion || device.androidVersion}
                  {" / "}
                  {device.platform}
                </p>
              </div>
              <button
                className="mt-1 grid h-6 w-6 shrink-0 place-items-center text-dim/50 opacity-0 transition hover:text-alarm group-hover:opacity-100"
                title="Forget device"
                type="button"
                onClick={() => void actions.forgetDevice(device.id)}
              >
                <Trash2 size={13} />
              </button>
            </div>
          ))
        )}
      </div>

      <div className="border-t border-line-mid px-4 py-3">
        <StatusPill status={devices.some((d) => d.status === "connected") ? "connected" : "offline"} />
        <p className="mt-2 font-mono text-[10px] leading-relaxed text-dim/70">
          Identity via Discord. Pairing records live in Supabase.
        </p>
      </div>
    </aside>
  );
}

function platformLabel(platform: string) {
  if (platform === "ios") return "iOS";
  if (platform === "web") return "Web";
  if (platform === "android") return "Android";
  return "Mobile";
}
