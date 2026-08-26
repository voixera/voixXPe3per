import Foundation
import ReplayKit
import CoreMedia

open class SampleHandler: RPBroadcastSampleHandler {
    private var socket: DesktopSocket?
    private var encoder: H264Encoder?
    private var isConnected = false
    private var encoderReady = false

    override open func broadcastStarted(withSetupInfo setupInfo: [String: NSObject]?) {
        let store = TrustedDesktopStore()
        guard let trusted = store.load(), let secret = store.trustSecret() else {
            finishBroadcastWithError(NSError(domain: "voiXPe3per", code: 1, userInfo: [NSLocalizedDescriptionKey: "Desktop belum di-pair — buka app voiXPe3per dan scan QR dulu"]))
            return
        }

        socket = DesktopSocket()
        let h264 = H264Encoder { [weak self] frameData, isKeyframe in
            guard let self, self.isConnected else { return }
            let packet = FramePacket.wrap(encoded: frameData, keyFrame: isKeyframe)
            self.socket?.sendBinary(packet)
        }
        encoder = h264

        socket?.onEvent = { [weak self] json in
            guard let type = json["type"] as? String else { return }
            if type == "stream.request_keyframe" {
                self?.encoder?.requestKeyFrame()
            }
        }

        // The encoder is prepared on the first video frame so the stream uses
        // the REAL screen dimensions/orientation — UIScreen is not reliable
        // inside a broadcast extension (no UI context).
        socket?.reconnect(trusted: trusted, trustSecret: secret) { [weak self] result in
            switch result {
            case .success:
                self?.isConnected = true
            case .failure(let err):
                self?.finishBroadcastWithError(err)
            }
        }
    }

    override open func broadcastPaused() {}
    override open func broadcastResumed() {}

    override open func broadcastFinished() {
        encoder?.stop()
        socket?.close()
        isConnected = false
    }

    override open func processSampleBuffer(_ sampleBuffer: CMSampleBuffer, with sampleBufferType: RPSampleBufferType) {
        guard sampleBufferType == .video, isConnected else { return }

        if !encoderReady {
            guard let formatDesc = CMSampleBufferGetFormatDescription(sampleBuffer) else { return }
            var width: Int32 = 0
            var height: Int32 = 0
            CMVideoFormatDescriptionGetDimensions(formatDesc, width: &width, height: &height)

            guard let encoder else {
                finishBroadcastWithError(NSError(domain: "voiXPe3per", code: 3, userInfo: [NSLocalizedDescriptionKey: "Encoder tidak tersedia"]))
                return
            }
            guard encoder.prepare(width: width, height: height, fps: 60) else {
                finishBroadcastWithError(NSError(domain: "voiXPe3per", code: 4, userInfo: [NSLocalizedDescriptionKey: "H264 encoder gagal dimulai (\(width)x\(height))"]))
                return
            }
            socket?.sendText("{\"type\":\"stream.start\",\"codec\":\"H264\",\"width\":\(width),\"height\":\(height),\"targetFps\":60}")
            encoderReady = true
        }

        encoder?.encode(sampleBuffer: sampleBuffer)
    }
}
