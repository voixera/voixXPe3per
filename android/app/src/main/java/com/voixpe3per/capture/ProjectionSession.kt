package com.voixpe3per.capture

import android.content.Context
import android.content.Intent
import android.media.projection.MediaProjection
import android.media.projection.MediaProjectionManager

class ProjectionSession(private val context: Context) {
    fun create(resultCode: Int, data: Intent): MediaProjection {
        val manager = context.getSystemService(MediaProjectionManager::class.java)
        return manager.getMediaProjection(resultCode, data)
    }
}
