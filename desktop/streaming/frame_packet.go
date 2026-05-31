package streaming

import (
	"encoding/binary"
	"errors"
)

const (
	frameHeaderSize = 12
	frameMagicA     = byte('V')
	frameMagicB     = byte('X')
	frameVersion    = byte(1)

	flagKeyFrame = byte(1)
)

type ParsedFrame struct {
	KeyFrame    bool
	TimestampNS int64
	Payload     []byte
}

func ParseFramePacket(packet []byte) (ParsedFrame, error) {
	if len(packet) < frameHeaderSize {
		return ParsedFrame{}, errors.New("frame packet too small")
	}
	if packet[0] != frameMagicA || packet[1] != frameMagicB || packet[2] != frameVersion {
		return ParsedFrame{}, errors.New("invalid frame packet header")
	}

	timestamp := int64(binary.BigEndian.Uint64(packet[4:12]))
	return ParsedFrame{
		KeyFrame:    packet[3]&flagKeyFrame != 0,
		TimestampNS: timestamp,
		Payload:     packet[12:],
	}, nil
}
