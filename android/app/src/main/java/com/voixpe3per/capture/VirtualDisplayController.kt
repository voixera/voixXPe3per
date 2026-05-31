package com.voixpe3per.capture

import android.hardware.display.DisplayManager
import android.hardware.display.VirtualDisplay
import android.media.projection.MediaProjection
import android.view.Surface

class VirtualDisplayController {
    private var display: VirtualDisplay? = null

    fun start(
        projection: MediaProjection,
        width: Int,
        height: Int,
        densityDpi: Int,
        surface: Surface
    ) {
        display = projection.createVirtualDisplay(
            "voiXPe3per-screen",
            width,
            height,
            densityDpi,
            DisplayManager.VIRTUAL_DISPLAY_FLAG_AUTO_MIRROR,
            surface,
            null,
            null
        )
    }

    fun stop() {
        display?.release()
        display = null
    }
}
