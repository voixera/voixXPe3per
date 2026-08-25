package com.voixpe3per.network

import android.util.Log
import okhttp3.WebSocket

class FrameSender(private val socket: WebSocket) {
    @Volatile
    private var droppedSinceKeyframe = 0

    fun sendStreamStart(width: Int, height: Int, fps: Int) {
        val ok = socket.send(
            """
            {"type":"stream.start","codec":"H264","width":$width,"height":$height,"targetFps":$fps}
            """.trimIndent()
        )
        logIfRejected("stream.start", ok)
    }

    fun send(encoded: ByteArray, keyFrame: Boolean) {
        // OkHttp buffers internally; send() returns false once the peer or the
        // queue is gone — ignoring it silently swallowed every later frame.
        val ok = socket.send(FramePacket.wrap(encoded, keyFrame))
        if (!ok) {
            droppedSinceKeyframe += 1
            Log.w(TAG, "frame dropped (queue closed) total=$droppedSinceKeyframe bytes=${encoded.size}")
            return
        }
        if (keyFrame) {
            droppedSinceKeyframe = 0
        }
    }

    private fun logIfRejected(what: String, ok: Boolean) {
        if (!ok) {
            Log.w(TAG, "$what rejected by websocket")
        }
    }

    companion object {
        private const val TAG = "voiXPe3per"
    }
}
