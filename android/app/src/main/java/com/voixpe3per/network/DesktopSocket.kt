package com.voixpe3per.network

import com.voixpe3per.pairing.PairingPayload
import com.voixpe3per.security.LocalDevice
import com.voixpe3per.security.TrustedDesktop
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.WebSocket
import okhttp3.WebSocketListener
import org.json.JSONObject
import java.util.concurrent.TimeUnit

class DesktopSocket {
    private val client = OkHttpClient.Builder()
        .pingInterval(10, TimeUnit.SECONDS)
        .retryOnConnectionFailure(true)
        .build()

    fun pair(
        payload: PairingPayload,
        identity: LocalDevice,
        onPaired: (TrustedDesktop, WebSocket) -> Unit,
        onError: (String) -> Unit
    ) {
        val url = payload.url
        if (url.isBlank()) {
            onError("QR harus memakai WSS publik")
            return
        }

        val request = Request.Builder()
            .url(url)
            .build()

        client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                joinRelayIfNeeded(webSocket, payload.mode, payload.room)
                webSocket.send(pairMessage(payload, identity).toString())
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                val json = JSONObject(text)
                when (json.optString("type")) {
                    "pair.success" -> {
                        val trusted = TrustedDesktop(
                            mode = payload.mode,
                            host = payload.host,
                            port = payload.port,
                            relay = payload.relay,
                            publicUrl = payload.publicUrl,
                            room = payload.room,
                            deviceId = json.getString("deviceId"),
                            trustSecret = json.getString("trustSecret")
                        )
                        onPaired(trusted, webSocket)
                    }
                    "stream.request_keyframe" -> SocketRegistry.requestKeyframe()
                    "pair.failed", "error" -> onError(json.optString("message", "pairing failed"))
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                onError(t.message ?: "websocket error")
            }
        })
    }

    fun reconnect(
        trusted: TrustedDesktop,
        onConnected: (WebSocket) -> Unit,
        onError: (String) -> Unit
    ) {
        val url = trusted.url
        if (url.isBlank()) {
            onError("trusted desktop belum punya WSS publik")
            return
        }

        val request = Request.Builder()
            .url(url)
            .build()

        client.newWebSocket(request, object : WebSocketListener() {
            override fun onOpen(webSocket: WebSocket, response: Response) {
                joinRelayIfNeeded(webSocket, trusted.mode, trusted.room)
                val message = JSONObject()
                    .put("type", "device.reconnect")
                    .put("deviceId", trusted.deviceId)
                    .put("trustSecret", trusted.trustSecret)
                webSocket.send(message.toString())
            }

            override fun onMessage(webSocket: WebSocket, text: String) {
                val json = JSONObject(text)
                when (json.optString("type")) {
                    "reconnect.success" -> onConnected(webSocket)
                    "stream.request_keyframe" -> SocketRegistry.requestKeyframe()
                    "reconnect.failed", "error" -> onError(json.optString("message", "reconnect failed"))
                }
            }

            override fun onFailure(webSocket: WebSocket, t: Throwable, response: Response?) {
                onError(t.message ?: "websocket error")
            }
        })
    }

    private fun pairMessage(payload: PairingPayload, identity: LocalDevice): JSONObject {
        val device = JSONObject()
            .put("id", identity.id)
            .put("name", identity.name)
            .put("model", identity.model)
            .put("manufacturer", identity.manufacturer)
            .put("platform", identity.platform)
            .put("osName", identity.osName)
            .put("osVersion", identity.osVersion)
            .put("androidVersion", identity.androidVersion)

        val capabilities = JSONObject()
            .put("encoder", "h264")
            .put("maxFps", 60)

        return JSONObject()
            .put("type", "pair.verify")
            .put("token", payload.token)
            .put("room", payload.room)
            .put("device", device)
            .put("capabilities", capabilities)
    }

    private fun joinRelayIfNeeded(webSocket: WebSocket, mode: String, room: String) {
        if (mode != "relay") {
            return
        }

        webSocket.send(
            JSONObject()
                .put("type", "relay.join")
                .put("role", "android")
                .put("room", room)
                .toString()
        )
    }
}
