package com.voixpe3per

import android.Manifest
import android.app.Activity
import android.content.Intent
import android.content.pm.PackageManager
import android.media.projection.MediaProjectionManager
import android.os.Build
import android.os.Bundle
import android.view.Gravity
import android.widget.Button
import android.widget.LinearLayout
import android.widget.TextView
import androidx.appcompat.app.AppCompatActivity
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat
import com.google.zxing.integration.android.IntentIntegrator
import com.voixpe3per.capture.ScreenCaptureService
import com.voixpe3per.pairing.PairingPayload
import com.voixpe3per.pairing.PairingRepository
import com.voixpe3per.security.DeviceIdentity
import com.voixpe3per.security.TrustedDeviceStore

class MainActivity : AppCompatActivity() {
    private val captureRequest = 7101
    private lateinit var statusText: TextView
    private lateinit var pairingRepository: PairingRepository
    private lateinit var trustedStore: TrustedDeviceStore

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        trustedStore = TrustedDeviceStore(this)
        pairingRepository = PairingRepository(DeviceIdentity(this), trustedStore)
        requestNotificationPermission()
        renderUi()
        attemptReconnect()
    }

    private fun renderUi() {
        val root = LinearLayout(this).apply {
            orientation = LinearLayout.VERTICAL
            gravity = Gravity.CENTER
            setPadding(42, 42, 42, 42)
            setBackgroundColor(0xFF0F1013.toInt())
        }

        val title = TextView(this).apply {
            text = "voiXPe3per"
            textSize = 26f
            setTextColor(0xFFE6EBF2.toInt())
            gravity = Gravity.CENTER
        }
        val subtitle = TextView(this).apply {
            text = "Scan QR dari desktop untuk mulai mirror layar lewat LAN."
            textSize = 14f
            setTextColor(0xFF8D98A8.toInt())
            gravity = Gravity.CENTER
            setPadding(0, 12, 0, 34)
        }
        val scanButton = Button(this).apply {
            text = "Scan QR Pairing"
            setOnClickListener { startQrScanner() }
        }
        val reconnectButton = Button(this).apply {
            text = "Reconnect Trusted Desktop"
            setOnClickListener { attemptReconnect() }
        }
        statusText = TextView(this).apply {
            text = "Status: waiting"
            textSize = 13f
            setTextColor(0xFF4FD18B.toInt())
            gravity = Gravity.CENTER
            setPadding(0, 28, 0, 0)
        }

        root.addView(title)
        root.addView(subtitle)
        root.addView(scanButton)
        root.addView(reconnectButton)
        root.addView(statusText)
        setContentView(root)
    }

    private fun startQrScanner() {
        IntentIntegrator(this)
            .setDesiredBarcodeFormats(IntentIntegrator.QR_CODE)
            .setPrompt("Scan QR voiXPe3per desktop")
            .setBeepEnabled(false)
            .setOrientationLocked(false)
            .initiateScan()
    }

    private fun attemptReconnect() {
        val trusted = trustedStore.load()
        if (trusted == null) {
            statusText.text = "Status: belum ada trusted desktop"
            return
        }

        statusText.text = "Status: reconnecting ${trusted.host}:${trusted.port}"
        pairingRepository.reconnectTrusted(
            onReady = {
                statusText.text = "Status: trusted desktop connected"
                requestCapturePermission()
            },
            onError = { message ->
                statusText.text = "Status: reconnect failed - $message"
            }
        )
    }

    private fun requestCapturePermission() {
        val manager = getSystemService(MediaProjectionManager::class.java)
        startActivityForResult(manager.createScreenCaptureIntent(), captureRequest)
    }

    private fun startCapture(resultCode: Int, data: Intent) {
        val intent = Intent(this, ScreenCaptureService::class.java).apply {
            putExtra(ScreenCaptureService.EXTRA_RESULT_CODE, resultCode)
            putExtra(ScreenCaptureService.EXTRA_RESULT_DATA, data)
        }
        ContextCompat.startForegroundService(this, intent)
    }

    override fun onActivityResult(requestCode: Int, resultCode: Int, data: Intent?) {
        val scan = IntentIntegrator.parseActivityResult(requestCode, resultCode, data)
        if (scan != null) {
            if (scan.contents != null) {
                pair(scan.contents)
            } else {
                statusText.text = "Status: QR scan cancelled"
            }
            return
        }

        if (requestCode == captureRequest && resultCode == Activity.RESULT_OK && data != null) {
            startCapture(resultCode, data)
            statusText.text = "Status: streaming layar"
            return
        }

        super.onActivityResult(requestCode, resultCode, data)
    }

    private fun pair(rawQr: String) {
        val payload = runCatching { PairingPayload.fromQr(rawQr) }.getOrElse {
            statusText.text = "Status: QR tidak valid"
            return
        }

        statusText.text = "Status: pairing ${payload.host}:${payload.port}"
        pairingRepository.pair(
            payload = payload,
            onReady = {
                statusText.text = "Status: pairing sukses"
                requestCapturePermission()
            },
            onError = { message ->
                statusText.text = "Status: pairing gagal - $message"
            }
        )
    }

    private fun requestNotificationPermission() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU &&
            ActivityCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) != PackageManager.PERMISSION_GRANTED
        ) {
            ActivityCompat.requestPermissions(this, arrayOf(Manifest.permission.POST_NOTIFICATIONS), 32)
        }
    }
}
