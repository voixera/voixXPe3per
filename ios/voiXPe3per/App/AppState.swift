import Foundation

@MainActor
final class AppState: ObservableObject {
    @Published var status = "Ready to pair"
    @Published var trustedDesktop: TrustedDesktop?
    @Published var isPairing = false

    private let identity = DeviceIdentity()
    private let store = TrustedDesktopStore()
    private let socket = DesktopSocket()

    init() {
        trustedDesktop = store.load()
        if trustedDesktop != nil {
            status = "Trusted desktop saved"
        }
    }

    func pair(raw: String) {
        do {
            let payload = try PairingPayload.parse(raw)
            isPairing = true
            status = "Connecting to desktop..."
            socket.pair(payload: payload, identity: identity.snapshot()) { [weak self] result in
                Task { @MainActor in
                    self?.isPairing = false
                    switch result {
                    case .success(let pairResult):
                        self?.store.save(pairResult)
                        self?.trustedDesktop = pairResult.desktop
                        self?.status = "Paired with desktop"
                    case .failure(let error):
                        self?.status = error.localizedDescription
                    }
                }
            }
        } catch {
            status = error.localizedDescription
        }
    }

    func reconnect() {
        guard let trustedDesktop else {
            status = "No trusted desktop"
            return
        }

        isPairing = true
        status = "Reconnecting..."
        guard let trustSecret = store.trustSecret() else {
            isPairing = false
            status = "Trust secret missing"
            return
        }

        socket.reconnect(trusted: trustedDesktop, trustSecret: trustSecret) { [weak self] result in
            Task { @MainActor in
                self?.isPairing = false
                switch result {
                case .success:
                    self?.status = "Connected"
                case .failure(let error):
                    self?.status = error.localizedDescription
                }
            }
        }
    }

    func forgetDesktop() {
        store.clear()
        trustedDesktop = nil
        socket.close()
        status = "Trusted desktop removed"
    }
}
