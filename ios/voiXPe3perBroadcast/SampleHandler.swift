import Foundation
import ReplayKit
import CoreMedia

open class SampleHandler: RPBroadcastSampleHandler {
    private var socket: DesktopSocket?
    private var encoder: H264Encoder?
    private var isConnected = false

    override open func broadcastStarted(withSetupInfo setupInfo: [String: NSObject]?) {
        let store = TrustedDesktopStore()
        guard let trusted = store.load(), let secret = store.trustSecret() else {
            finishBroadcastWithError(NSError(domain: "voiXPe3per", code: 1, userInfo: [NSLocalizedDescriptionKey: "Desktop belum di-pair"]))
            return
        }

        socket = DesktopSocket()
        encoder = H264Encoder { [weak self] frameData, isKeyframe in
            guard let self, self.isConnected else { return }
            let packet = FramePacket.wrap(encoded: frameData, keyFrame: isKeyframe)
            self.socket?.sendBinary(packet)
        }

        socket?.reconnect(trusted: trusted, trustSecret: secret) { [weak self] result in
            switch result {
            case .success:
                self?.isConnected = true
                self?.socket?.sendText("{\"type\":\"stream.start\",\"codec\":\"H264\",\"width\":1080,\"height\":1920,\"targetFps\":60}")
                self?.encoder?.prepare(width: 1080, height: 1920, fps: 60)
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
        encoder?.encode(sampleBuffer: sampleBuffer)
    }
}
