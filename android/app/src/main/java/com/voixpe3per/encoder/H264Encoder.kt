package com.voixpe3per.encoder

import android.media.MediaCodec
import android.media.MediaCodecInfo
import android.media.MediaFormat
import android.view.Surface
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
                if (index >= 0) {
                    val buffer = mediaCodec.getOutputBuffer(index)
                    if (buffer != null && info.size > 0) {
                        val bytes = ByteArray(info.size)
                        buffer.position(info.offset)
                        buffer.limit(info.offset + info.size)
                        buffer.get(bytes)
                        val keyFrame = info.flags and MediaCodec.BUFFER_FLAG_KEY_FRAME != 0
                        onFrame(bytes, keyFrame)
                    }
                    mediaCodec.releaseOutputBuffer(index, false)
                }
            }
        }
    }
}
