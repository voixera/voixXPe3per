package com.voixpe3per.network

import okhttp3.WebSocket

object SocketRegistry {
    @Volatile
    private var socket: WebSocket? = null

    fun attach(webSocket: WebSocket) {
        socket = webSocket
    }

    fun current(): WebSocket? = socket

    fun detach() {
        socket?.close(1000, "closed")
        socket = null
    }
}
