import { DevicePanel } from "./components/DevicePanel";
import { PairingPanel } from "./components/PairingPanel";
import { StreamStage } from "./components/StreamStage";
import { TopBar } from "./components/TopBar";
import { useAppState } from "./store/appStore";

export function App() {
  const { state } = useAppState();
  const connectedDevice = state.devices.find((device) => device.status === "connected");

  return (
    <main className="h-screen min-h-[620px] overflow-hidden bg-shell-950 text-slate-200">
      <TopBar connectedDevice={connectedDevice} metrics={state.metrics} />
      <div className="grid h-[calc(100vh-54px)] grid-cols-[250px_1fr]">
        <DevicePanel devices={state.devices} />
        <section className="flex min-w-0 flex-col border-l border-black/40 bg-shell-900">
          {connectedDevice ? (
            <StreamStage device={connectedDevice} frame={state.latestFrame} metrics={state.metrics} />
          ) : (
            <PairingPanel pairing={state.pairing} booting={state.booting} />
          )}
        </section>
      </div>
    </main>
  );
}
