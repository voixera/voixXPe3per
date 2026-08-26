import Foundation
import VideoToolbox
import CoreMedia

public final class H264Encoder {
    private var session: VTCompressionSession?
    private let onEncoded: (Data, Bool) -> Void
    private var width: Int32 = 0
    private var height: Int32 = 0
    // Benign race: set from socket thread, read on encode path; worst case the
    // forced keyframe lands one frame later.
    private var pendingKeyFrameRequest = false

    public init(onEncoded: @escaping (Data, Bool) -> Void) {
        self.onEncoded = onEncoded
    }

    public func prepare(width: Int32, height: Int32, fps: Int32 = 60, bitrate: Int32 = 2_500_000) -> Bool {
        self.width = width
        self.height = height
        stop()

        let status = VTCompressionSessionCreate(
            allocator: kCFAllocatorDefault,
            width: width,
            height: height,
            codecType: kCMVideoCodecType_H264,
            encoderSpecification: nil,
            imageBufferAttributes: nil,
            compressedDataAllocator: nil,
            outputCallback: compressionCallback,
            refcon: Unmanaged.passUnretained(self).toOpaque(),
            compressionSessionOut: &session
        )

        guard status == noErr, let session else { return false }

        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_RealTime, value: kCFBooleanTrue)
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_ProfileLevel, value: kVTProfileLevel_H264_Baseline_AutoLevel)
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_AverageBitRate, value: NSNumber(value: bitrate))
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_ExpectedFrameRate, value: NSNumber(value: fps))
        VTSessionSetProperty(session, key: kVTCompressionPropertyKey_MaxKeyFrameInterval, value: NSNumber(value: fps * 2))
        guard VTCompressionSessionPrepareToEncodeFrames(session) == noErr else {
            stop()
            return false
        }
        return true
    }

    public func requestKeyFrame() {
        pendingKeyFrameRequest = true
    }

    public func encode(sampleBuffer: CMSampleBuffer) {
        guard let session, let imageBuffer = CMSampleBufferGetImageBuffer(sampleBuffer) else { return }
        let pts = CMSampleBufferGetPresentationTimeStamp(sampleBuffer)
        let duration = CMSampleBufferGetDuration(sampleBuffer)
        var frameProps: CFDictionary?
        if pendingKeyFrameRequest {
            frameProps = [kVTEncodeFrameOptionKey_ForceKeyFrame: true] as CFDictionary
            pendingKeyFrameRequest = false
        }
        VTCompressionSessionEncodeFrame(
            session,
            imageBuffer: imageBuffer,
            presentationTimeStamp: pts,
            duration: duration,
            frameProperties: frameProps,
            sourceFrameRefcon: nil,
            infoFlagsOut: nil
        )
    }

    public func stop() {
        if let session {
            VTCompressionSessionInvalidate(session)
            self.session = nil
        }
    }

    private func handleEncodedSampleBuffer(_ sampleBuffer: CMSampleBuffer) {
        guard let formatDesc = CMSampleBufferGetFormatDescription(sampleBuffer),
              let dataBuffer = CMSampleBufferGetDataBuffer(sampleBuffer) else { return }

        var isKeyframe = false
        if let attachments = CMSampleBufferGetSampleAttachmentsArray(sampleBuffer, createIfNecessary: false) as? [[CFString: Any]],
           let first = attachments.first {
            isKeyframe = !(first[kCMSampleAttachmentKey_NotSync] as? Bool ?? false)
        }

        var payload = Data()
        let naluHeader = Data([0x00, 0x00, 0x00, 0x01])

        if isKeyframe {
            var count = 0
            CMVideoFormatDescriptionGetH264ParameterSetAtIndex(formatDesc, parameterSetIndex: 0, parameterSetPointerOut: nil, parameterSetSizeOut: nil, parameterSetCountOut: &count, nalUnitHeaderLengthOut: nil)
            for i in 0..<count {
                var ptr: UnsafePointer<UInt8>?
                var size = 0
                if CMVideoFormatDescriptionGetH264ParameterSetAtIndex(formatDesc, parameterSetIndex: i, parameterSetPointerOut: &ptr, parameterSetSizeOut: &size, parameterSetCountOut: nil, nalUnitHeaderLengthOut: nil) == noErr,
                   let ptr, size > 0 {
                    payload.append(naluHeader)
                    payload.append(ptr, count: size)
                }
            }
        }

        var lengthAtOffset = 0
        var totalLength = 0
        var dataPointer: UnsafeMutablePointer<CChar>?
        if CMBlockBufferGetDataPointer(dataBuffer, atOffset: 0, lengthAtOffsetOut: &lengthAtOffset, totalLengthOut: &totalLength, dataPointerOut: &dataPointer) == noErr,
           let dataPointer {
            var offset = 0
            while offset < totalLength - 4 {
                var naluLen: UInt32 = 0
                memcpy(&naluLen, dataPointer.advanced(by: offset), 4)
                naluLen = CFSwapInt32BigToHost(naluLen)
                offset += 4
                if offset + Int(naluLen) <= totalLength {
                    payload.append(naluHeader)
                    let rawUnit = UnsafeRawPointer(dataPointer.advanced(by: offset)).assumingMemoryBound(to: UInt8.self)
                    payload.append(rawUnit, count: Int(naluLen))
                    offset += Int(naluLen)
                } else {
                    break
                }
            }
        }

        if !payload.isEmpty {
            onEncoded(payload, isKeyframe)
        }
    }
}

private func compressionCallback(
    outputCallbackRefCon: UnsafeMutableRawPointer?,
    sourceFrameRefCon: UnsafeMutableRawPointer?,
    status: OSStatus,
    infoFlags: VTEncodeInfoFlags,
    sampleBuffer: CMSampleBuffer?
) {
    guard status == noErr, let sampleBuffer, let refCon = outputCallbackRefCon else { return }
    let encoder = Unmanaged<H264Encoder>.fromOpaque(refCon).takeUnretainedValue()
    encoder.perform(#selector(encoderProxyHandle(_:)), with: sampleBuffer)
}

extension H264Encoder: NSObjectProtocol {
    public func isEqual(_ object: Any?) -> Bool { self === object as AnyObject? }
    public var hash: Int { ObjectIdentifier(self).hashValue }
    public var superclass: AnyClass? { nil }
    public func `self`() -> Self { self }
    public func perform(_ aSelector: Selector!) -> Unmanaged<AnyObject>! { nil }
    public func perform(_ aSelector: Selector!, with object: Any!) -> Unmanaged<AnyObject>! {
        if aSelector == #selector(encoderProxyHandle(_:)), let buf = object as? CMSampleBuffer {
            handleEncodedSampleBuffer(buf)
        }
        return nil
    }
    public func perform(_ aSelector: Selector!, with object1: Any!, with object2: Any!) -> Unmanaged<AnyObject>! { nil }
    public func isProxy() -> Bool { false }
    public func isKind(of aClass: AnyClass) -> Bool { false }
    public func isMember(of aClass: AnyClass) -> Bool { false }
    public func conforms(to aProtocol: Protocol) -> Bool { false }
    public func responds(to aSelector: Selector!) -> Bool { aSelector == #selector(encoderProxyHandle(_:)) }
    public var description: String { "H264Encoder" }

    @objc func encoderProxyHandle(_ buffer: Any) {}
}
