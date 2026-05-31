package com.voixpe3per.network

import okhttp3.WebSocket

class FrameSender(private val socket: WebSocket) {
    fun sendStreamStart(width: Int, height: Int, fps: Int) {
        socket.send(
            """
            {"type":"stream.start","codec":"H264","width":$width,"height":$height,"targetFps":$fps}
            """.trimIndent()
        )
    }

    fun send(encoded: ByteArray, keyFrame: Boolean) {
        socket.send(FramePacket.wrap(encoded, keyFrame))
    }
}
