package com.voixpe3per.network

import okio.ByteString
import java.nio.ByteBuffer
import java.nio.ByteOrder

object FramePacket {
    private const val headerSize = 12
    private const val keyFrameFlag = 1

    fun wrap(encoded: ByteArray, keyFrame: Boolean, timestampNs: Long = System.currentTimeMillis() * 1_000_000L): ByteString {
        val buffer = ByteBuffer.allocate(headerSize + encoded.size).order(ByteOrder.BIG_ENDIAN)
        buffer.put('V'.code.toByte())
        buffer.put('X'.code.toByte())
        buffer.put(1)
        buffer.put(if (keyFrame) keyFrameFlag.toByte() else 0)
        buffer.putLong(timestampNs)
        buffer.put(encoded)
        return ByteString.of(*buffer.array())
    }
}
