package com.voixpe3per.encoder

class BitrateController(
    private val minBitrate: Int = 2_000_000,
    private val maxBitrate: Int = 10_000_000
) {
    fun targetBitrate(width: Int, height: Int, fps: Int): Int {
        val pixels = width * height
        val estimate = (pixels * fps * 0.08f).toInt()
        return estimate.coerceIn(minBitrate, maxBitrate)
    }
}
