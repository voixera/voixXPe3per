import type { CamFrame } from "../types";

// ponytail: loads supabase-js from CDN (see index.html); bundle it locally if
// the desktop app must run fully offline. Upgrade path: npm i @supabase/supabase-js.
type SupabaseClientLike = {
  channel: (name: string, opts?: unknown) => {
    on: (event: string, filter: unknown, cb: (msg: { payload: CamFrame }) => void) => unknown;
    subscribe: (cb: (status: string) => void) => void;
    unsubscribe: () => Promise<"ok" | "timed out" | "error">;
  };
  removeAllChannels: () => void;
};

let client: SupabaseClientLike | null = null;

function getClient(url: string, anonKey: string): SupabaseClientLike {
  const factory = (window as unknown as { supabase?: { createClient: (u: string, k: string) => SupabaseClientLike } }).supabase;
  if (!factory) {
    throw new Error("supabase-js failed to load");
  }
  if (!client) {
    client = factory.createClient(url, anonKey);
  }
  return client;
}

// subscribeCam listens to the pairing room's camera broadcast. Runs inside
// the webview (a real browser), which Cloudflare accepts — unlike raw Go
// websocket dials, which get 403'd by bot management.
export function subscribeCam(
  supabaseUrl: string,
  anonKey: string,
  room: string,
  onFrame: (frame: CamFrame) => void,
  onStatus?: (status: string) => void
): () => void {
  let channel: ReturnType<SupabaseClientLike["channel"]> | null = null;
  try {
    const c = getClient(supabaseUrl, anonKey);
    channel = c.channel(`room:${room}`, { config: { broadcast: { self: false } } });
    channel.on("broadcast", { event: "cam" }, (msg) => {
      if (msg?.payload?.j) {
        onFrame(msg.payload);
      }
    });
    channel.subscribe((status) => {
      onStatus?.(status);
    });
  } catch (err) {
    onStatus?.("CLIENT_ERROR " + (err instanceof Error ? err.message : String(err)));
  }

  return () => {
    void channel?.unsubscribe();
  };
}
