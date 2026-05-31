package com.voixpe3per.network

import okhttp3.WebSocket

object SocketRegistry {
    @Volatile
    private var socket: WebSocket? = null
    @Volatile
    private var requestKeyframe: (() -> Unit)? = null

    fun attach(webSocket: WebSocket) {
        socket = webSocket
    }

    fun current(): WebSocket? = socket

    fun setKeyframeRequester(request: (() -> Unit)?) {
        requestKeyframe = request
    }

    fun requestKeyframe() {
        requestKeyframe?.invoke()
    }

    fun detach() {
        socket?.close(1000, "closed")
        socket = null
        requestKeyframe = null
    }
}
