import Foundation

struct PairingPayload: Equatable {
    let mode: String
    let host: String
    let port: Int
    let token: String
    let relay: String
    let publicURL: String
    let room: String

    func webSocketURL() throws -> URL {
        if mode == "relay" || mode == "direct" {
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

    static func parse(_ raw: String) throws -> PairingPayload {
        let trimmed = raw.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmed.hasPrefix("http://") || trimmed.hasPrefix("https://") || trimmed.hasPrefix("voixpe3per://") {
            return try parseURL(trimmed)
        }

        guard let data = trimmed.data(using: .utf8) else {
            throw PairingError.invalidPayload
        }

        let object = try JSONSerialization.jsonObject(with: data)
        guard let json = object as? [String: Any] else {
            throw PairingError.invalidPayload
        }

        return PairingPayload(
            mode: json.string("mode") ?? "lan",
            host: json.string("host") ?? "",
            port: json.int("port") ?? 8080,
            token: json.string("token") ?? "",
            relay: json.string("relay") ?? "",
            publicURL: json.string("public") ?? "",
            room: json.string("room") ?? ""
        )
    }

    private static func parseURL(_ raw: String) throws -> PairingPayload {
        guard let components = URLComponents(string: raw) else {
            throw PairingError.invalidPayload
        }

        return PairingPayload(
            mode: components.queryValue("mode") ?? "relay",
            host: components.queryValue("host") ?? "",
            port: Int(components.queryValue("port") ?? "") ?? 8080,
            token: components.queryValue("token") ?? "",
            relay: components.queryValue("relay") ?? "",
            publicURL: components.queryValue("public") ?? "",
            room: components.queryValue("room") ?? ""
        )
    }
}

enum PairingError: LocalizedError {
    case invalidPayload
    case invalidWebSocketURL
    case serverRejected(String)

    var errorDescription: String? {
        switch self {
        case .invalidPayload:
            return "QR pairing tidak valid"
        case .invalidWebSocketURL:
            return "WebSocket URL tidak valid"
        case .serverRejected(let message):
            return message
        }
    }
}

private extension URLComponents {
    func queryValue(_ name: String) -> String? {
        queryItems?.first { $0.name == name }?.value
    }
}

private extension Dictionary where Key == String, Value == Any {
    func string(_ key: String) -> String? {
        self[key] as? String
    }

    func int(_ key: String) -> Int? {
        if let value = self[key] as? Int {
            return value
        }
        if let value = self[key] as? String {
            return Int(value)
        }
        return nil
    }
}
