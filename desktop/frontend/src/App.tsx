import { useState } from "react";
import { DevicePanel } from "./components/DevicePanel";
import { LoginGate } from "./components/LoginGate";
import { PairingPanel } from "./components/PairingPanel";
import { StreamStage } from "./components/StreamStage";
import { TopBar } from "./components/TopBar";
import { useAppState } from "./store/appStore";

export function App() {
  const { state, actions } = useAppState();
  const [pairingOpen, setPairingOpen] = useState(false);
  const connectedDevice = state.devices.find((device) => device.status === "connected");

  if (!state.auth.loggedIn) {
    return <LoginGate />;
  }

  return (
    <main className="flex h-screen min-h-[620px] flex-col overflow-hidden bg-ink-900 text-bone">
      <TopBar auth={state.auth} metrics={state.metrics} />
      <div className="grid min-h-0 flex-1 grid-cols-[260px_1fr]">
        <DevicePanel
          devices={state.devices}
          onAddDevice={() => {
            void actions.startFreshPairing();
            setPairingOpen(true);
          }}
        />
        <section className="flex min-w-0 flex-col border-l border-line-mid bg-ink-850">
          {connectedDevice && !pairingOpen ? (
            <StreamStage device={connectedDevice} frame={state.latestFrame} metrics={state.metrics} />
          ) : (
            <PairingPanel
              pairing={state.pairing}
              booting={state.booting}
              onClose={connectedDevice ? () => setPairingOpen(false) : undefined}
            />
          )}
        </section>
      </div>
    </main>
  );
}
