import http from "node:http";
import https from "node:https";
import { URL } from "node:url";
import crypto from "node:crypto";
import { WebSocketServer } from "ws";

const port = Number.parseInt(process.env.PORT ?? "8090", 10);
const DISCORD_CLIENT_ID = process.env.DISCORD_CLIENT_ID || "123456789012345678";
const DISCORD_CLIENT_SECRET = process.env.DISCORD_CLIENT_SECRET || "";
const JWT_SECRET = process.env.JWT_SECRET || "voixpe3per-cloud-super-secret-key";

// Cloud Store in memory (or extensible to Redis/Postgres)
const rooms = new Map(); // roomCode -> Set<WebSocket>
const userSessions = new Map(); // token -> { id, username, discriminator, avatar, email }
const userRooms = new Map(); // userId -> active roomCode

const server = http.createServer(async (request, response) => {
  const reqUrl = new URL(request.url, `http://${request.headers.host}`);
  
  // CORS Headers
  response.setHeader("Access-Control-Allow-Origin", "*");
  response.setHeader("Access-Control-Allow-Methods", "GET, POST, OPTIONS");
  response.setHeader("Access-Control-Allow-Headers", "Content-Type, Authorization");

  if (request.method === "OPTIONS") {
    response.writeHead(204);
    response.end();
    return;
  }

  // Health check
  if (reqUrl.pathname === "/health") {
    response.writeHead(200, { "content-type": "application/json" });
    response.end(JSON.stringify({ status: "ok", service: "voiXPe3per Cloud Relay", timestamp: Date.now() }));
    return;
  }

  // Discord OAuth URL Generator
  if (reqUrl.pathname === "/api/auth/discord/url") {
    const redirectUri = reqUrl.searchParams.get("redirect_uri") || "https://voixxpe3per.vercel.app/pair";
    const state = crypto.randomBytes(16).toString("hex");
    const scope = encodeURIComponent("identify email");
    const discordAuthUrl = `https://discord.com/api/oauth2/authorize?client_id=${DISCORD_CLIENT_ID}&redirect_uri=${encodeURIComponent(redirectUri)}&response_type=code&scope=${scope}&state=${state}`;
    
    response.writeHead(200, { "content-type": "application/json" });
    response.end(JSON.stringify({ url: discordAuthUrl, state }));
    return;
  }

  // Discord OAuth Callback / Code Exchange
  if (reqUrl.pathname === "/api/auth/discord/callback" && request.method === "POST") {
    let body = "";
    request.on("data", chunk => body += chunk);
    request.on("end", async () => {
      try {
        const { code, redirect_uri } = JSON.parse(body || "{}");
        if (!code) {
          response.writeHead(400, { "content-type": "application/json" });
          response.end(JSON.stringify({ error: "Code is required" }));
          return;
        }

        // Exchange code with Discord API
        const tokenData = await exchangeDiscordCode(code, redirect_uri);
        if (!tokenData.access_token) {
          response.writeHead(401, { "content-type": "application/json" });
          response.end(JSON.stringify({ error: "Discord authentication failed", details: tokenData }));
          return;
        }

        // Fetch User profile from Discord
        const profile = await fetchDiscordUser(tokenData.access_token);
        const sessionToken = generateSessionToken(profile);
        userSessions.set(sessionToken, profile);

        response.writeHead(200, { "content-type": "application/json" });
        response.end(JSON.stringify({
          token: sessionToken,
          user: {
            id: profile.id,
            username: profile.username,
            globalName: profile.global_name || profile.username,
            avatar: profile.avatar ? `https://cdn.discordapp.com/avatars/${profile.id}/${profile.avatar}.png` : null,
            email: profile.email
          }
        }));
      } catch (err) {
        response.writeHead(500, { "content-type": "application/json" });
        response.end(JSON.stringify({ error: err.message }));
      }
    });
    return;
  }

  // User Profile
  if (reqUrl.pathname === "/api/user/me") {
    const authHeader = request.headers.authorization;
    const token = authHeader?.replace(/^Bearer\s+/i, "");
    const user = userSessions.get(token);
    if (!user) {
      response.writeHead(401, { "content-type": "application/json" });
      response.end(JSON.stringify({ error: "Unauthorized" }));
      return;
    }

    response.writeHead(200, { "content-type": "application/json" });
    response.end(JSON.stringify({ user }));
    return;
  }

  response.writeHead(404);
  response.end();
});

const wss = new WebSocketServer({ server, path: "/ws" });

wss.on("connection", (socket) => {
  socket.room = "";
  socket.role = "";
  socket.user = null;

  socket.on("message", (data, isBinary) => {
    if (!isBinary && maybeHandleControl(socket, data)) {
      return;
    }

    if (!socket.room) {
      socket.send(JSON.stringify({ type: "relay.error", message: "join a room first" }));
      return;
    }

    const peers = rooms.get(socket.room) ?? new Set();
    for (const peer of peers) {
      if (peer !== socket && peer.readyState === peer.OPEN) {
        peer.send(data, { binary: isBinary });
      }
    }
  });

  socket.on("close", () => {
    leaveRoom(socket);
  });
});

function maybeHandleControl(socket, data) {
  let message;
  try {
    message = JSON.parse(data.toString("utf8"));
  } catch {
    return false;
  }

  if (message.type === "auth.session") {
    const user = userSessions.get(message.token);
    if (user) {
      socket.user = user;
      socket.send(JSON.stringify({ type: "auth.success", user }));
    } else {
      socket.send(JSON.stringify({ type: "auth.failed", message: "Invalid session token" }));
    }
    return true;
  }

  if (message.type !== "relay.join") {
    return false;
  }

  const room = String(message.room ?? "").trim().toUpperCase();
  const role = String(message.role ?? "").trim().toLowerCase();
  if (!room || !["desktop", "android", "ios", "web"].includes(role)) {
    socket.send(JSON.stringify({ type: "relay.error", message: "invalid relay.join" }));
    return true;
  }

  leaveRoom(socket);
  socket.room = room;
  socket.role = role;

  if (!rooms.has(room)) {
    rooms.set(room, new Set());
  }
  rooms.get(room).add(socket);

  socket.send(JSON.stringify({ type: "relay.ready", room, role }));
  broadcast(room, socket, { type: "relay.peer_joined", role, user: socket.user });
  return true;
}

function leaveRoom(socket) {
  if (!socket.room) {
    return;
  }

  const peers = rooms.get(socket.room);
  if (peers) {
    peers.delete(socket);
    if (peers.size === 0) {
      rooms.delete(socket.room);
    } else {
      broadcast(socket.room, socket, { type: "relay.peer_left", role: socket.role });
    }
  }

  socket.room = "";
  socket.role = "";
}

function broadcast(room, sender, payload) {
  const data = JSON.stringify(payload);
  const peers = rooms.get(room) ?? new Set();
  for (const peer of peers) {
    if (peer !== sender && peer.readyState === peer.OPEN) {
      peer.send(data);
    }
  }
}

function generateSessionToken(profile) {
  const header = Buffer.from(JSON.stringify({ alg: "HS256", typ: "JWT" })).toString("base64url");
  const payload = Buffer.from(JSON.stringify({ id: profile.id, u: profile.username, exp: Date.now() + 30 * 86400000 })).toString("base64url");
  const sign = crypto.createHmac("sha256", JWT_SECRET).update(`${header}.${payload}`).digest("base64url");
  return `${header}.${payload}.${sign}`;
}

async function exchangeDiscordCode(code, redirectUri) {
  if (!DISCORD_CLIENT_SECRET) {
    // Development fallback mock if not configured
    return { access_token: `mock_access_token_${code}` };
  }

  const params = new URLSearchParams();
  params.append("client_id", DISCORD_CLIENT_ID);
  params.append("client_secret", DISCORD_CLIENT_SECRET);
  params.append("grant_type", "authorization_code");
  params.append("code", code);
  params.append("redirect_uri", redirectUri);

  const res = await fetch("https://discord.com/api/oauth2/token", {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: params
  });
  return await res.json();
}

async function fetchDiscordUser(accessToken) {
  if (accessToken.startsWith("mock_access_token")) {
    return {
      id: "998877665544332211",
      username: "voix_user",
      global_name: "voiXPe3per Pilot",
      avatar: null,
      email: "user@voixpe3per.cloud"
    };
  }

  const res = await fetch("https://discord.com/api/users/@me", {
    headers: { Authorization: `Bearer ${accessToken}` }
  });
  return await res.json();
}

server.listen(port, () => {
  console.log(`voiXPe3per Cloud Relay + Discord Auth backend listening on :${port}`);
});
