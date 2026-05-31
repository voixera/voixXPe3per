package com.voixpe3per.pairing

import com.voixpe3per.network.DesktopSocket
import com.voixpe3per.network.SocketRegistry
import com.voixpe3per.security.DeviceIdentity
import com.voixpe3per.security.TrustedDeviceStore

class PairingRepository(
    private val identity: DeviceIdentity,
    private val trustedStore: TrustedDeviceStore
) {
    private val socket = DesktopSocket()

    fun pair(payload: PairingPayload, onReady: () -> Unit, onError: (String) -> Unit) {
        socket.pair(
            payload = payload,
            identity = identity.snapshot(),
            onPaired = { trusted, webSocket ->
                trustedStore.save(trusted)
                SocketRegistry.attach(webSocket)
                onReady()
            },
            onError = onError
        )
    }

    fun reconnectTrusted(onReady: () -> Unit, onError: (String) -> Unit) {
        val trusted = trustedStore.load() ?: run {
            onError("no trusted desktop")
            return
        }

        socket.reconnect(
            trusted = trusted,
            onConnected = { webSocket ->
                SocketRegistry.attach(webSocket)
                onReady()
            },
            onError = onError
        )
    }
}
