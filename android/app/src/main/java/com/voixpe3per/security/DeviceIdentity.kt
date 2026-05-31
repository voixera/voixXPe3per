package com.voixpe3per.security

import android.content.Context
import android.os.Build
import java.util.UUID

data class LocalDevice(
    val id: String,
    val name: String,
    val model: String,
    val manufacturer: String,
    val androidVersion: String
)

class DeviceIdentity(context: Context) {
    private val prefs = context.getSharedPreferences("voixpe3per_identity", Context.MODE_PRIVATE)

    fun snapshot(): LocalDevice {
        val id = prefs.getString("device_id", null) ?: UUID.randomUUID().toString().also {
            prefs.edit().putString("device_id", it).apply()
        }

        return LocalDevice(
            id = id,
            name = "${Build.MANUFACTURER} ${Build.MODEL}",
            model = Build.MODEL ?: "Android",
            manufacturer = Build.MANUFACTURER ?: "Android",
            androidVersion = Build.VERSION.RELEASE ?: "unknown"
        )
    }
}
