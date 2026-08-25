import Foundation

struct TrustedDesktop: Codable, Equatable {
    let mode: String
    let host: String
    let port: Int
    let relay: String
    let publicURL: String
    let room: String
    let deviceId: String

    func webSocketURL() throws -> URL {
        if mode == "relay" || mode == "direct" || mode == "cloud" {
            guard let relayURL = URL(string: relay.isEmpty ? publicURL : relay) else {
                throw PairingError.invalidWebSocketURL
            }
            return relayURL
        }
        guard let lanURL = URL(string: "ws://\(host):\(port)/ws") else {
            throw PairingError.invalidWebSocketURL
        }
        return lanURL
    }
}

// ponytail: trust secret lives in the shared App Group defaults (not the
// keychain) because keychain-access-groups need matching provisioning
// profiles on app + extension; App Group defaults are enough for dev.
// Upgrade path: add keychain-access-groups entitlements + kSecAttrAccessGroup.
enum AppGroup {
    static let id = "group.com.voixpe3per"
}

final class TrustedDesktopStore {
    private let metadataKey = "voixpe3per.trusted.desktop"
    private let secretKey = "voixpe3per.trust.secret"
    private let secretStore = KeychainSecretStore()
    // The broadcast extension runs in its own sandbox: plain UserDefaults and
    // its keychain are invisible there, which made every broadcast start fail
    // with "Desktop belum di-pair". Suite defaults are shared via App Group.
    private let defaults: UserDefaults

    init() {
        defaults = UserDefaults(suiteName: AppGroup.id) ?? .standard
    }

    func save(_ desktop: TrustedDesktop, trustSecret: String? = nil) {
        if let data = try? JSONEncoder().encode(desktop) {
            defaults.set(data, forKey: metadataKey)
        }
        if let trustSecret {
            defaults.set(trustSecret, forKey: secretKey)
            secretStore.set(trustSecret, forKey: secretKey)
        }
    }

    func save(_ pairResult: PairResult) {
        save(pairResult.desktop, trustSecret: pairResult.trustSecret)
    }

    func load() -> TrustedDesktop? {
        guard let data = defaults.data(forKey: metadataKey) else {
            return nil
        }
        return try? JSONDecoder().decode(TrustedDesktop.self, from: data)
    }

    func trustSecret() -> String? {
        if let shared = defaults.string(forKey: secretKey), !shared.isEmpty {
            return shared
        }
        return secretStore.get(secretKey)
    }

    func clear() {
        defaults.removeObject(forKey: metadataKey)
        defaults.removeObject(forKey: secretKey)
        secretStore.delete(secretKey)
    }
}

struct PairResult {
    let desktop: TrustedDesktop
    let trustSecret: String
}
