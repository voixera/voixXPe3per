import { LogOut, RefreshCw } from "lucide-react";
import { useAppState } from "../store/appStore";
import type { AuthIdentity, StreamMetrics } from "../types";

export function TopBar({ auth, metrics }: { auth: AuthIdentity; metrics: StreamMetrics }) {
  const { actions } = useAppState();

  return (
    <header className="flex h-[52px] items-stretch justify-between border-b border-line-mid bg-ink-900">
      <div className="flex items-center gap-4 border-r border-line-mid px-5">
        <div className="grid h-7 w-7 place-items-center border border-acid font-display text-sm font-bold text-acid">
          P
        </div>
        <div className="leading-none">
          <p className="font-display text-sm font-semibold uppercase tracking-[0.2em] text-bone">PeeperPhone</p>
          <p className="label-tech mt-1">Field Terminal v1</p>
        </div>
      </div>

      <div className="hidden items-center gap-8 px-6 font-mono text-[11px] uppercase tracking-[0.18em] text-dim md:flex">
        <Readout k="FPS" v={String(metrics.fps)} />
        <Readout k="Codec" v={metrics.codec || "H264"} />
        <Readout k="Link" v={metrics.transport || "WSS"} />
        <Readout k="Latency" v={`${metrics.latencyMs}ms`} />
      </div>

      <div className="flex items-stretch divide-x divide-line-mid border-l border-line-mid">
        {auth.loggedIn ? (
          <>
            <button className="btn-hard m-2 h-auto self-center border-0 px-3 py-1 hover:bg-transparent hover:text-acid" type="button" title="Refresh pairing" onClick={() => void actions.refreshPairing()}>
              <RefreshCw size={14} />
            </button>
            <div className="flex items-center gap-3 px-4">
              <span className="font-mono text-xs text-bone">{auth.name}</span>
            </div>
            <button
              className="grid w-12 place-items-center text-dim transition-colors hover:bg-alarm hover:text-ink-950"
              type="button"
              title="Sign out"
              onClick={() => void actions.logout()}
            >
              <LogOut size={15} />
            </button>
          </>
        ) : (
          <button className="btn-hard m-2 self-center" type="button" onClick={() => void actions.loginWithDiscord()}>
            Login
          </button>
        )}
      </div>
    </header>
  );
}

function Readout({ k, v }: { k: string; v: string }) {
  return (
    <span className="flex items-baseline gap-2">
      <span className="text-dim/70">{k}</span>
      <span className="text-acid">{v}</span>
    </span>
  );
}
