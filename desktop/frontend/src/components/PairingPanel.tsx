import { RefreshCw, Wifi } from "lucide-react";
import { useAppState } from "../store/appStore";
import type { PairingSession } from "../types";

export function PairingPanel({ pairing, booting }: { pairing: PairingSession; booting: boolean }) {
  const { actions } = useAppState();
  const transport = pairing.relayUrl || "Public WSS";

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-black/40 bg-shell-850 px-5 py-3">
        <div>
          <h2 className="text-sm font-semibold text-slate-100">PhoneMirror Pairing</h2>
          <p className="text-xs text-slate-500">{transport}</p>
        </div>
        <button
          className="toolbar-button"
          type="button"
          onClick={() => void actions.refreshPairing()}
        >
          <RefreshCw size={15} />
          Refresh QR
        </button>
      </div>

      <div className="grid flex-1 place-items-center bg-shell-900">
        <div className="pairing-shell animate-fade-in">
          <div className="mb-5 flex items-center justify-center gap-2 text-xs uppercase tracking-[0.18em] text-slate-500">
            <Wifi size={14} />
            {pairing.mode === "direct" ? "Direct Public WSS" : "Global Public Relay"}
          </div>
          <div className="mx-auto grid h-[330px] w-[330px] place-items-center border border-shell-600 bg-white p-4 shadow-2xl shadow-black/40">
            {pairing.qrDataUrl ? (
              <img className="h-full w-full object-contain" src={pairing.qrDataUrl} alt="Pairing QR Code" />
            ) : (
              <div className="h-full w-full animate-pulse bg-slate-200" />
            )}
          </div>
          <div className="mt-6 text-center">
            <h3 className="text-lg font-semibold text-slate-100">Scan QR lalu klik Izinkan pairing</h3>
            <p className="mt-2 text-sm text-slate-400">
              {pairing.mode === "direct"
                ? "Mode direct memakai WSS publik menuju desktop. Browser hanya pairing, bukan mirroring layar."
                : `Room ${pairing.room || "-"} melalui relay publik, tanpa harus satu jaringan.`}
            </p>
            <p className="mt-5 font-mono text-xs uppercase tracking-[0.18em] text-signal-amber">
              Status: {booting ? "Starting pairing transport..." : pairing.status}
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}
