package com.voixpe3per.pairing

import android.net.Uri
import org.json.JSONObject

data class PairingPayload(
    val mode: String,
    val host: String,
    val port: Int,
    val token: String,
    val relay: String,
    val publicUrl: String,
    val room: String
) {
    val url: String
        get() = if (mode == "relay" || mode == "direct") relay.ifBlank { publicUrl } else "ws://$host:$port/ws"

    companion object {
        fun fromQr(raw: String): PairingPayload {
            if (raw.startsWith("http://") || raw.startsWith("https://")) {
                return fromUrl(raw)
            }

            val json = JSONObject(raw)
            return PairingPayload(
                mode = json.optString("mode", "lan"),
                host = json.optString("host", ""),
                port = json.optInt("port", 8080),
                token = json.optString("token", ""),
                relay = json.optString("relay", ""),
                publicUrl = json.optString("public", ""),
                room = json.optString("room", "")
            )
        }

        private fun fromUrl(raw: String): PairingPayload {
            val uri = Uri.parse(raw)
            return PairingPayload(
                mode = uri.getQueryParameter("mode") ?: "relay",
                host = uri.getQueryParameter("host") ?: "",
                port = uri.getQueryParameter("port")?.toIntOrNull() ?: 8080,
                token = uri.getQueryParameter("token") ?: "",
                relay = uri.getQueryParameter("relay") ?: "",
                publicUrl = uri.getQueryParameter("public") ?: "",
                room = uri.getQueryParameter("room") ?: ""
            )
        }
    }
}
