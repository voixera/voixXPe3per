package com.voixpe3per.capture

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.media.projection.MediaProjection
import android.os.Build
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import android.util.DisplayMetrics
import android.view.WindowManager
import com.voixpe3per.encoder.H264Encoder
import com.voixpe3per.network.DesktopSocket
import com.voixpe3per.network.FrameSender
import com.voixpe3per.network.SocketRegistry
import com.voixpe3per.security.TrustedDeviceStore

class ScreenCaptureService : Service() {
    private val channelId = "voixpe3per_capture"
    private var projection: MediaProjection? = null
    private var projectionCallback: MediaProjection.Callback? = null
    private var encoder: H264Encoder? = null
    private val virtualDisplay = VirtualDisplayController()
    private val mainHandler = Handler(Looper.getMainLooper())
    private var stoppingCapture = false

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        startForeground(42, notification())
        val resultCode = intent?.getIntExtra(EXTRA_RESULT_CODE, 0) ?: return START_NOT_STICKY
        val resultData = intent.getParcelableExtra<Intent>(EXTRA_RESULT_DATA) ?: return START_NOT_STICKY
        connectAndStart(resultCode, resultData)
        return START_STICKY
    }

    private fun connectAndStart(resultCode: Int, resultData: Intent) {
        val trusted = TrustedDeviceStore(this).load() ?: return
        DesktopSocket().reconnect(
            trusted = trusted,
            onConnected = { socket ->
                SocketRegistry.attach(socket)
                startCapture(resultCode, resultData, FrameSender(socket))
            },
            onError = {
                SocketRegistry.current()?.let { startCapture(resultCode, resultData, FrameSender(it)) }
            }
        )
    }

    private fun startCapture(resultCode: Int, resultData: Intent, sender: FrameSender) {
        if (encoder != null) {
            return
        }

        val metrics = displayMetrics()
        val width = metrics.widthPixels.coerceAtMost(1080)
        val height = metrics.heightPixels.coerceAtMost(2400)
        val fps = 60

        val mediaProjection = ProjectionSession(this).create(resultCode, resultData)
        projection = mediaProjection
        registerProjectionCallback(mediaProjection)

        val h264Encoder = H264Encoder(width, height, fps) { bytes, keyFrame ->
            sender.send(bytes, keyFrame)
        }
        encoder = h264Encoder
        SocketRegistry.setKeyframeRequester { h264Encoder.requestKeyFrame() }

        val surface = h264Encoder.start()
        val started = runCatching {
            virtualDisplay.start(
                projection = mediaProjection,
                width = width,
                height = height,
                densityDpi = metrics.densityDpi,
                surface = surface
            )
        }.onFailure {
            stopCapture(releaseProjection = true)
            stopSelf()
        }.isSuccess

        if (!started) {
            return
        }

        sender.sendStreamStart(width, height, fps)
        h264Encoder.requestKeyFrame()
    }

    private fun registerProjectionCallback(mediaProjection: MediaProjection) {
        val callback = object : MediaProjection.Callback() {
            override fun onStop() {
                stopCapture(releaseProjection = false)
                stopSelf()
            }
        }
        projectionCallback = callback
        mediaProjection.registerCallback(callback, mainHandler)
    }

    private fun stopCapture(releaseProjection: Boolean) {
        if (stoppingCapture) {
            return
        }

        stoppingCapture = true
        try {
            virtualDisplay.stop()
            encoder?.stop()
            encoder = null
            SocketRegistry.setKeyframeRequester(null)

            val activeProjection = projection
            projectionCallback?.let { callback ->
                activeProjection?.let { runCatching { it.unregisterCallback(callback) } }
            }
            projectionCallback = null

            if (releaseProjection) {
                runCatching { activeProjection?.stop() }
            }
            projection = null
            SocketRegistry.detach()
        } finally {
            stoppingCapture = false
        }
    }

    override fun onDestroy() {
        stopCapture(releaseProjection = true)
        super.onDestroy()
    }

    private fun displayMetrics(): DisplayMetrics {
        val metrics = DisplayMetrics()
        val windowManager = getSystemService(WindowManager::class.java)
        @Suppress("DEPRECATION")
        windowManager.defaultDisplay.getRealMetrics(metrics)
        return metrics
    }

    private fun notification(): Notification {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(channelId, "voiXPe3per Capture", NotificationManager.IMPORTANCE_LOW)
            getSystemService(NotificationManager::class.java).createNotificationChannel(channel)
        }
        return Notification.Builder(this, channelId)
            .setContentTitle("voiXPe3per streaming")
            .setContentText("Screen mirroring aktif di jaringan lokal")
            .setSmallIcon(android.R.drawable.presence_video_online)
            .build()
    }

    companion object {
        const val EXTRA_RESULT_CODE = "result_code"
        const val EXTRA_RESULT_DATA = "result_data"
    }
}
