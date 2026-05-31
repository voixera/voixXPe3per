import http from "node:http";
import { WebSocketServer } from "ws";

const port = Number.parseInt(process.env.PORT ?? "8090", 10);
const rooms = new Map();

const server = http.createServer((request, response) => {
  if (request.url === "/health") {
    response.writeHead(200, { "content-type": "text/plain" });
    response.end("voiXPe3per relay ok");
    return;
  }

  response.writeHead(404);
  response.end();
});

const wss = new WebSocketServer({ server, path: "/ws" });

wss.on("connection", (socket) => {
  socket.room = "";
  socket.role = "";

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
  broadcast(room, socket, { type: "relay.peer_joined", role });
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

server.listen(port, () => {
  console.log(`voiXPe3per relay listening on :${port}`);
});
