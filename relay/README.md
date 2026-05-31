# voiXPe3per Public WSS Relay

Relay ini harus berjalan di host yang mendukung WebSocket persisten. Untuk flow tanpa app native, halaman Vercel `/pair` akan connect ke relay ini memakai `wss://.../ws`.

Default desktop mengarah ke:

```text
wss://voixpe3per-relay.onrender.com/ws
```

## Deploy ke Render

1. Push repo ke GitHub.
2. Di Render, pilih `New` -> `Blueprint`.
3. Pilih repo `voixera/voixXPe3per`.
4. Render akan membaca `render.yaml` di root.
5. Setelah service `voixpe3per-relay` live, endpoint WebSocket publiknya:

```text
wss://voixpe3per-relay.onrender.com/ws
```

Health check:

```text
https://voixpe3per-relay.onrender.com/health
```

## Local dev saja

```bash
npm install
npm start
```

Local dev memakai `ws://127.0.0.1:8090/ws`, tetapi build desktop production sekarang tidak memakai URL lokal sebagai default.
