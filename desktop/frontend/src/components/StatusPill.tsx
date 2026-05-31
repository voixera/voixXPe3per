import type { DeviceStatus } from "../types";

export function StatusPill({ status }: { status: DeviceStatus }) {
  const connected = status === "connected";
  return (
    <span className="inline-flex items-center gap-2 text-[11px] uppercase tracking-[0.16em] text-slate-300">
      <span className={`h-2 w-2 rounded-full ${connected ? "bg-signal-green" : "border border-slate-500"}`} />
      {connected ? "Connected" : "Offline"}
    </span>
  );
}
