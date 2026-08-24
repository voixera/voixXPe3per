import { RefreshCw } from "lucide-react";
import { useAppState } from "../store/appStore";
import type { PairingSession } from "../types";
import { Brackets } from "./Brackets";

const STEPS = [
  "Open the camera on your phone",
  "Scan the pairing code",
  "Approve with your Discord account"
];

export function PairingPanel({ pairing, booting }: { pairing: PairingSession; booting: boolean }) {
  const { actions } = useAppState();
  const isCloud = pairing.mode === "cloud";

  return (
    <div className="flex h-full flex-col">
      <div className="flex items-center justify-between border-b border-line-mid bg-ink-900 px-5 py-2.5">
        <p className="label-tech">01 / Pairing</p>
        <button className="btn-hard h-7 px-3" type="button" onClick={() => void actions.refreshPairing()}>
          <RefreshCw size={13} />
          New Code
        </button>
      </div>

      <div className="grid flex-1 place-items-center overflow-y-auto bg-ink-900 p-6">
        <div className="rise-in w-full max-w-[560px]">
          <div className="panel scan-sweep p-8">
            <Brackets />
            <div className="mx-auto grid h-[320px] w-[320px] place-items-center border border-line-mid bg-white p-3">
              {pairing.qrDataUrl ? (
                <img className="h-full w-full object-contain" src={pairing.qrDataUrl} alt="Pairing code" />
              ) : (
                <div className="h-full w-full animate-pulse bg-line-dim" />
              )}
            </div>

            <div className="mt-6 flex items-center justify-between gap-4">
              <div>
                <p className="label-tech">Room</p>
                <p className="font-mono text-lg tracking-[0.3em] text-acid">{pairing.room || "--------"}</p>
              </div>
              <div className="text-right">
                <p className="label-tech">Channel</p>
                <p className="font-mono text-xs uppercase text-bone">
                  {isCloud ? "Supabase Cloud" : pairing.mode === "direct" ? "Direct WSS" : "Public Relay"}
                </p>
              </div>
            </div>
          </div>

          <ol className="mt-6 space-y-0 border border-line-mid">
            {STEPS.map((step, index) => (
              <li
                key={step}
                className={`flex items-baseline gap-4 px-4 py-2.5 font-mono text-xs text-dim ${
                  index > 0 ? "border-t border-line-dim" : ""
                }`}
              >
                <span className="text-acid">0{index + 1}</span>
                {step}
              </li>
            ))}
          </ol>

          <p className="mt-4 flex items-center font-mono text-[11px] uppercase tracking-[0.22em] text-amber">
            Status:{booting ? " starting transport" : ` ${pairing.status}`}
            <span className="cursor-blink" />
          </p>
        </div>
      </div>
    </div>
  );
}
