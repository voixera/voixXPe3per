package com.voixpe3per.security

import android.content.Context

data class TrustedDesktop(
    val mode: String,
    val host: String,
    val port: Int,
    val relay: String,
    val publicUrl: String,
    val room: String,
    val deviceId: String,
    val trustSecret: String
) {
    val url: String
        get() = relay.ifBlank { publicUrl }
}

class TrustedDeviceStore(context: Context) {
    private val prefs = context.getSharedPreferences("voixpe3per_trusted_desktop", Context.MODE_PRIVATE)
    private val protector = SecretProtector()

    fun save(desktop: TrustedDesktop) {
        prefs.edit()
            .putString("mode", desktop.mode)
            .putString("host", desktop.host)
            .putInt("port", desktop.port)
            .putString("relay", desktop.relay)
            .putString("public", desktop.publicUrl)
            .putString("room", desktop.room)
            .putString("device_id", desktop.deviceId)
            .putString("trust_secret", protector.encrypt(desktop.trustSecret))
            .apply()
    }

    fun load(): TrustedDesktop? {
        val mode = prefs.getString("mode", "relay") ?: "relay"
        val host = prefs.getString("host", "") ?: ""
        val relay = prefs.getString("relay", "") ?: ""
        val public = prefs.getString("public", "") ?: ""
        if (relay.isBlank() && public.isBlank()) {
            return null
        }
        val deviceId = prefs.getString("device_id", null) ?: return null
        val trustSecret = prefs.getString("trust_secret", null)?.let { protector.decrypt(it) } ?: return null
        return TrustedDesktop(
            mode = mode,
            host = host,
            port = prefs.getInt("port", 8080),
            relay = relay,
            publicUrl = public,
            room = prefs.getString("room", "") ?: "",
            deviceId = deviceId,
            trustSecret = trustSecret
        )
    }

    fun clear() {
        prefs.edit().clear().apply()
    }
}
