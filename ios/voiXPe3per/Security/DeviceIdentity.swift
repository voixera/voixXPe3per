import UIKit

struct LocalDevice: Codable {
    let id: String
    let name: String
    let model: String
    let manufacturer: String
    let platform: String
    let osName: String
    let osVersion: String
}

struct DeviceIdentity {
    private let deviceIDKey = "voixpe3per.device.id"

    func snapshot() -> LocalDevice {
        let defaults = UserDefaults.standard
        let id = defaults.string(forKey: deviceIDKey) ?? UUID().uuidString
        defaults.set(id, forKey: deviceIDKey)

        let device = UIDevice.current
        return LocalDevice(
            id: id,
            name: device.name,
            model: device.model,
            manufacturer: "Apple",
            platform: "ios",
            osName: device.systemName,
            osVersion: device.systemVersion
        )
    }
}
