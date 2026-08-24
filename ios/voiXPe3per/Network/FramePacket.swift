import Foundation

public struct FramePacket {
    public static let headerSize = 12
    public static let keyFrameFlag: UInt8 = 1

    public static func wrap(encoded: Data, keyFrame: Bool, timestampNs: UInt64 = UInt64(Date().timeIntervalSince1970 * 1_000_000_000)) -> Data {
        var packet = Data(capacity: headerSize + encoded.count)
        packet.append(contentsOf: [UInt8(ascii: "V"), UInt8(ascii: "X"), 1, keyFrame ? keyFrameFlag : 0])
        var beTimestamp = timestampNs.bigEndian
        withUnsafeBytes(of: &beTimestamp) { packet.append(contentsOf: $0) }
        packet.append(encoded)
        return packet
    }
}
