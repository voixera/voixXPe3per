import Foundation

struct TrustedDesktop: Codable, Equatable {
    let mode: String
    let host: String
    let port: Int
    let relay: String
    let room: String
    let deviceId: String

    func webSocketURL() throws -> URL {
        if mode == "relay", let relayURL = URL(string: relay) {
            return relayURL
        }
        guard let lanURL = URL(string: "ws://\(host):\(port)/ws") else {
            throw PairingError.invalidWebSocketURL
        }
        return lanURL
    }
}

final class TrustedDesktopStore {
    private let metadataKey = "voixpe3per.trusted.desktop"
    private let secretKey = "voixpe3per.trust.secret"
    private let secretStore = KeychainSecretStore()

    func save(_ desktop: TrustedDesktop, trustSecret: String? = nil) {
        let data = try? JSONEncoder().encode(desktop)
        UserDefaults.standard.set(data, forKey: metadataKey)
        if let trustSecret {
            secretStore.set(trustSecret, forKey: secretKey)
        }
    }

    func save(_ pairResult: PairResult) {
        save(pairResult.desktop, trustSecret: pairResult.trustSecret)
    }

    func load() -> TrustedDesktop? {
        guard let data = UserDefaults.standard.data(forKey: metadataKey) else {
            return nil
        }
        return try? JSONDecoder().decode(TrustedDesktop.self, from: data)
    }

    func trustSecret() -> String? {
        secretStore.get(secretKey)
    }

    func clear() {
        UserDefaults.standard.removeObject(forKey: metadataKey)
        secretStore.delete(secretKey)
    }
}

struct PairResult {
    let desktop: TrustedDesktop
    let trustSecret: String
}
