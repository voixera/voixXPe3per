package com.voixpe3per.capture

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Intent
import android.media.projection.MediaProjection
import android.os.Build
import android.os.IBinder
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
    private var encoder: H264Encoder? = null
    private val virtualDisplay = VirtualDisplayController()

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

        sender.sendStreamStart(width, height, fps)
        projection = ProjectionSession(this).create(resultCode, resultData)
        encoder = H264Encoder(width, height, fps) { bytes, keyFrame ->
            sender.send(bytes, keyFrame)
        }

        val surface = encoder?.start() ?: return
        virtualDisplay.start(
            projection = projection ?: return,
            width = width,
            height = height,
            densityDpi = metrics.densityDpi,
            surface = surface
        )
    }

    override fun onDestroy() {
        virtualDisplay.stop()
        encoder?.stop()
        projection?.stop()
        SocketRegistry.detach()
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
