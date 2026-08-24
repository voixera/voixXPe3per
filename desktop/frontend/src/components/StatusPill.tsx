import type { DeviceStatus } from "../types";

export function StatusPill({ status }: { status: DeviceStatus }) {
  const connected = status === "connected";
  return (
    <span className="inline-flex items-center gap-2 font-mono text-[10px] uppercase tracking-[0.22em] text-dim">
      <span className={`led ${connected ? "on" : "off"}`} />
      {connected ? "Live" : "Offline"}
    </span>
  );
}
