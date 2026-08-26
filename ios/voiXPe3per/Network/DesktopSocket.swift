import Foundation

final class DesktopSocket {
    private let session = URLSession(configuration: .default)
    private var task: URLSessionWebSocketTask?
    private var pingTimer: Timer?
    // Events arriving after the handshake (e.g. stream.request_keyframe).
    var onEvent: (([String: Any]) -> Void)?

    func pair(
        payload: PairingPayload,
        identity: LocalDevice,
        completion: @escaping (Result<PairResult, Error>) -> Void
    ) {
        do {
            let url = try payload.webSocketURL()
            let webSocket = session.webSocketTask(with: url)
            task = webSocket
            webSocket.resume()
            startPing()

            if payload.mode == "relay" || payload.mode == "cloud" {
                sendJSON(["type": "relay.join", "role": "ios", "room": payload.room], using: webSocket)
            }
            sendJSON(pairMessage(payload: payload, identity: identity), using: webSocket)
            receivePairResult(payload: payload, using: webSocket, completion: completion)
        } catch {
            completion(.failure(error))
        }
    }

    func reconnect(
        trusted: TrustedDesktop,
        trustSecret: String,
        completion: @escaping (Result<Void, Error>) -> Void
    ) {
        do {
            let url = try trusted.webSocketURL()
            let webSocket = session.webSocketTask(with: url)
            task = webSocket
            webSocket.resume()
            startPing()

            if trusted.mode == "relay" || trusted.mode == "cloud" {
                sendJSON(["type": "relay.join", "role": "ios", "room": trusted.room], using: webSocket)
            }
            sendJSON([
                "type": "device.reconnect",
                "deviceId": trusted.deviceId,
                "trustSecret": trustSecret
            ], using: webSocket)
            receiveReconnect(using: webSocket, completion: completion)
        } catch {
            completion(.failure(error))
        }
    }

    func sendText(_ text: String) {
        task?.send(.string(text)) { _ in }
    }

    func sendBinary(_ data: Data) {
        task?.send(.data(data)) { _ in }
    }

    func close() {
        pingTimer?.invalidate()
        pingTimer = nil
        task?.cancel(with: .normalClosure, reason: nil)
        task = nil
    }

    // Keepalive: relays drop idle sockets; a 20s ping keeps the broadcast
    // extension's connection alive for long silent stretches.
    private func startPing() {
        pingTimer?.invalidate()
        pingTimer = Timer.scheduledTimer(withTimeInterval: 20, repeats: true) { [weak self] _ in
            self?.task?.sendPing { _ in }
        }
    }

    private func startReceiveLoop(using webSocket: URLSessionWebSocketTask) {
        webSocket.receive { [weak self] result in
            guard let self else { return }
            switch result {
            case .failure:
                return
            case .success(let message):
                if let json = self.decode(message) {
                    self.onEvent?(json)
                }
                self.startReceiveLoop(using: webSocket)
            }
        }
    }

    private func receivePairResult(
        payload: PairingPayload,
        using webSocket: URLSessionWebSocketTask,
        completion: @escaping (Result<PairResult, Error>) -> Void
    ) {
        webSocket.receive { [weak self] result in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let message):
                guard let json = self?.decode(message) else {
                    self?.receivePairResult(payload: payload, using: webSocket, completion: completion)
                    return
                }

                switch json["type"] as? String {
                case "pair.success":
                    guard let deviceId = json["deviceId"] as? String,
                          let trustSecret = json["trustSecret"] as? String else {
                        completion(.failure(PairingError.invalidPayload))
                        return
                    }
                    let desktop = TrustedDesktop(
                        mode: payload.mode,
                        host: payload.host,
                        port: payload.port,
                        relay: payload.relay,
                        publicURL: payload.publicURL,
                        room: payload.room,
                        deviceId: deviceId
                    )
                    completion(.success(PairResult(desktop: desktop, trustSecret: trustSecret)))
                    // Keep the loop alive for post-pair control messages.
                    startReceiveLoop(using: webSocket)
                case "pair.failed", "error":
                    completion(.failure(PairingError.serverRejected(json["message"] as? String ?? "Pairing failed")))
                default:
                    self?.receivePairResult(payload: payload, using: webSocket, completion: completion)
                }
            }
        }
    }

    private func receiveReconnect(
        using webSocket: URLSessionWebSocketTask,
        completion: @escaping (Result<Void, Error>) -> Void
    ) {
        webSocket.receive { [weak self] result in
            switch result {
            case .failure(let error):
                completion(.failure(error))
            case .success(let message):
                guard let json = self?.decode(message) else {
                    self?.receiveReconnect(using: webSocket, completion: completion)
                    return
                }

                switch json["type"] as? String {
                case "reconnect.success":
                    completion(.success(()))
                    // Keep reading: desktop may send stream.request_keyframe.
                    startReceiveLoop(using: webSocket)
                case "reconnect.failed", "error":
                    completion(.failure(PairingError.serverRejected(json["message"] as? String ?? "Reconnect failed")))
                default:
                    self?.receiveReconnect(using: webSocket, completion: completion)
                }
            }
        }
    }

    private func pairMessage(payload: PairingPayload, identity: LocalDevice) -> [String: Any] {
        [
            "type": "pair.verify",
            "token": payload.token,
            "room": payload.room,
            "device": [
                "id": identity.id,
                "name": identity.name,
                "model": identity.model,
                "manufacturer": identity.manufacturer,
                "platform": identity.platform,
                "osName": identity.osName,
                "osVersion": identity.osVersion
            ],
            "capabilities": [
                "encoder": "h264",
                "maxFps": 60
            ]
        ]
    }

    private func sendJSON(_ object: [String: Any], using webSocket: URLSessionWebSocketTask) {
        guard JSONSerialization.isValidJSONObject(object),
              let data = try? JSONSerialization.data(withJSONObject: object),
              let text = String(data: data, encoding: .utf8) else {
            return
        }
        webSocket.send(.string(text)) { _ in }
    }

    private func decode(_ message: URLSessionWebSocketTask.Message) -> [String: Any]? {
        let data: Data?
        switch message {
        case .string(let text):
            data = text.data(using: .utf8)
        case .data(let payload):
            data = payload
        @unknown default:
            data = nil
        }

        guard let data,
              let object = try? JSONSerialization.jsonObject(with: data),
              let json = object as? [String: Any] else {
            return nil
        }
        return json
    }
}
