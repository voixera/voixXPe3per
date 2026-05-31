package com.voixpe3per.encoder

import android.media.MediaCodec
import android.media.MediaCodecInfo
import android.media.MediaFormat
import android.os.Bundle
import android.view.Surface
import java.nio.ByteBuffer
import java.util.concurrent.atomic.AtomicBoolean
import kotlin.concurrent.thread

class H264Encoder(
    private val width: Int,
    private val height: Int,
    private val fps: Int,
    private val onFrame: (bytes: ByteArray, keyFrame: Boolean) -> Unit
) {
    private val running = AtomicBoolean(false)
    private var codec: MediaCodec? = null
    private var inputSurface: Surface? = null
    @Volatile
    private var codecConfig = ByteArray(0)

    fun start(): Surface {
        val format = MediaFormat.createVideoFormat(MediaFormat.MIMETYPE_VIDEO_AVC, width, height).apply {
            setInteger(MediaFormat.KEY_COLOR_FORMAT, MediaCodecInfo.CodecCapabilities.COLOR_FormatSurface)
            setInteger(MediaFormat.KEY_BIT_RATE, BitrateController().targetBitrate(width, height, fps))
            setInteger(MediaFormat.KEY_FRAME_RATE, fps)
            setInteger(MediaFormat.KEY_I_FRAME_INTERVAL, 1)
            setInteger(MediaFormat.KEY_LATENCY, 0)
        }

        val mediaCodec = MediaCodec.createEncoderByType(MediaFormat.MIMETYPE_VIDEO_AVC)
        mediaCodec.configure(format, null, null, MediaCodec.CONFIGURE_FLAG_ENCODE)
        inputSurface = mediaCodec.createInputSurface()
        mediaCodec.start()
        codec = mediaCodec
        running.set(true)
        drain(mediaCodec)
        return inputSurface ?: error("encoder surface unavailable")
    }

    fun requestKeyFrame() {
        val parameters = Bundle().apply {
            putInt(MediaCodec.PARAMETER_KEY_REQUEST_SYNC_FRAME, 0)
        }
        runCatching { codec?.setParameters(parameters) }
    }

    fun stop() {
        running.set(false)
        runCatching { codec?.signalEndOfInputStream() }
        runCatching { inputSurface?.release() }
        runCatching { codec?.stop() }
        runCatching { codec?.release() }
        codec = null
        inputSurface = null
    }

    private fun drain(mediaCodec: MediaCodec) {
        thread(name = "voiXPe3per-h264-drain", isDaemon = true) {
            val info = MediaCodec.BufferInfo()
            while (running.get()) {
                val index = mediaCodec.dequeueOutputBuffer(info, 10_000)
                when {
                    index >= 0 -> {
                        val bytes = readOutput(mediaCodec, index, info)
                        if (bytes != null && bytes.isNotEmpty()) {
                            val isCodecConfig = info.flags and MediaCodec.BUFFER_FLAG_CODEC_CONFIG != 0
                            val keyFrame = info.flags and MediaCodec.BUFFER_FLAG_KEY_FRAME != 0
                            if (isCodecConfig) {
                                codecConfig = bytes
                            } else {
                                onFrame(attachCodecConfig(bytes, keyFrame), keyFrame)
                            }
                        }
                        mediaCodec.releaseOutputBuffer(index, false)
                    }
                    index == MediaCodec.INFO_OUTPUT_FORMAT_CHANGED -> {
                        codecConfig = readCodecConfig(mediaCodec.outputFormat)
                    }
                }
            }
        }
    }

    private fun readOutput(mediaCodec: MediaCodec, index: Int, info: MediaCodec.BufferInfo): ByteArray? {
        val buffer = mediaCodec.getOutputBuffer(index) ?: return null
        val duplicate = buffer.duplicate()
        duplicate.position(info.offset)
        duplicate.limit(info.offset + info.size)
        return copyRemaining(duplicate)
    }

    private fun readCodecConfig(format: MediaFormat): ByteArray {
        val parts = listOf("csd-0", "csd-1", "csd-2").mapNotNull { key ->
            if (format.containsKey(key)) {
                format.getByteBuffer(key)?.let { copyRemaining(it.duplicate()) }
            } else {
                null
            }
        }.filter { it.isNotEmpty() }

        return concat(parts)
    }

    private fun attachCodecConfig(bytes: ByteArray, keyFrame: Boolean): ByteArray {
        val config = codecConfig
        if (!keyFrame || config.isEmpty() || startsWith(bytes, config)) {
            return bytes
        }
        return concat(listOf(config, bytes))
    }

    private fun copyRemaining(buffer: ByteBuffer): ByteArray {
        val bytes = ByteArray(buffer.remaining())
        buffer.get(bytes)
        return bytes
    }

    private fun startsWith(bytes: ByteArray, prefix: ByteArray): Boolean {
        if (bytes.size < prefix.size) {
            return false
        }
        for (index in prefix.indices) {
            if (bytes[index] != prefix[index]) {
                return false
            }
        }
        return true
    }

    private fun concat(parts: List<ByteArray>): ByteArray {
        val size = parts.sumOf { it.size }
        val output = ByteArray(size)
        var offset = 0
        for (part in parts) {
            part.copyInto(output, destinationOffset = offset)
            offset += part.size
        }
        return output
    }
}
