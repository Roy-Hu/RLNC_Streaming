package qrlnc

import (
	"encoding/binary"
	"fmt"
)

// 8192 bits = 1024 bytes
var PKTBITNUM int = 8192
var PKTNUM int = PKTBITNUM / 64
var RNGSEED int64 = int64(1)

var TYPE_NORM byte = 0x1
var TYPE_XNC byte = 0x2
var TYPESIZE int = 1
var IDSIZE int = 4
var FILESIZESIZE int = 4

var CHUNKSIZE int = 1 << 17
var SYMBOLNUM int = (CHUNKSIZE * 8) / PKTBITNUM
var COEFNUM int = SYMBOLNUM / 64
var FRAMESIZE int = TYPESIZE + IDSIZE + FILESIZESIZE + COEFNUM*8 + PKTNUM*8

// XNC pkt
// Type + ChunkId + FileSize+ Packet

func EncodeXNCPkt(data XNC) ([]byte, error) {
	if len(data.Coefficient) != COEFNUM || len(data.Packet) != PKTNUM {
		return nil, fmt.Errorf("XNC pkt size is not correct")
	}

	pkt := []byte{}

	pkt = append(pkt, data.Type)

	chunkId := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkId, uint32(data.ChunkId))
	pkt = append(pkt, chunkId...)

	filesize := make([]byte, 4)
	binary.BigEndian.PutUint32(filesize, uint32(data.FileSize))
	pkt = append(pkt, filesize...)

	for i := 0; i < COEFNUM; i++ {
		coef := make([]byte, 8)
		binary.BigEndian.PutUint64(coef, uint64(data.Coefficient[i]))
		pkt = append(pkt, coef...)
	}

	for i := 0; i < PKTNUM; i++ {
		packet := make([]byte, 8)
		binary.BigEndian.PutUint64(packet, uint64(data.Packet[i]))
		pkt = append(pkt, packet...)
	}

	return pkt, nil
}

func DecodeXNCPkt(data []byte) (XNC, error) {
	if len(data) != FRAMESIZE {
		return XNC{}, fmt.Errorf("XNC pkt size is not correct")
	}

	xnc := XNC{}
	xnc.Type = data[0]
	xnc.ChunkId = int(binary.BigEndian.Uint32(data[1:5]))
	xnc.FileSize = int(binary.BigEndian.Uint32(data[5:9]))

	for i := 0; i < COEFNUM; i++ {
		xnc.Coefficient = append(xnc.Coefficient, binary.BigEndian.Uint64(data[9+i*8:17+i*8]))
	}

	for i := 0; i < PKTNUM; i++ {
		xnc.Packet = append(xnc.Packet, binary.BigEndian.Uint64(data[9+COEFNUM*8+i*8:17+COEFNUM*8+i*8]))
	}

	return xnc, nil
}

type XNC struct {
	Type        byte
	ChunkId     int
	FileSize    int
	Coefficient []uint64
	Packet      []uint64
}

// packBinaryBytesToUint64s takes a slice of bytes, where each byte is expected to be 0x00 or 0x01,
// and packs these into a slice of uint64 values, with each bit in the uint64 representing a byte from the input.
func PackBinaryBytesToUint64s(binaryBytes []byte) ([]uint64, int) {
	var result []uint64
	var currentUint64 uint64
	var bitPosition uint

	for _, byteVal := range binaryBytes {
		// Check if the byte is 0x01 and set the corresponding bit in currentUint64.
		if byteVal == 0x01 {
			currentUint64 |= 1 << bitPosition
		}

		bitPosition++

		// If we've filled up a uint64 or reached the end of the input, append to result and reset for the next uint64.
		if bitPosition == 64 {
			result = append(result, currentUint64)
			currentUint64 = 0
			bitPosition = 0
		}
	}

	// Handle any remaining bits that didn't fill up a final uint64.
	if bitPosition > 0 {
		result = append(result, currentUint64)
	}

	return result, len(binaryBytes)
}

// unpackUint64sToBinaryBytes takes a slice of uint64 values and an original length of binary data,
// and unpacks these into a slice of bytes, with each bit in the uint64 becoming a byte in the output,
// either 0x00 or 0x01, stopping once the original length is reached.
func UnpackUint64sToBinaryBytes(uint64s []uint64, originalLength int) []byte {
	var result []byte
	totalBits := 0 // Track the total number of bits unpacked.

	for _, uint64Val := range uint64s {
		for bitPosition := 0; bitPosition < 64; bitPosition++ {
			if totalBits == originalLength {
				// Stop unpacking if we've reached the original length.
				break
			}

			// Extract the bit at bitPosition from uint64Val.
			bit := (uint64Val >> bitPosition) & 1

			// Convert the bit to a byte (0x00 for 0, 0x01 for 1) and append it to the result.
			if bit == 1 {
				result = append(result, 0x01)
			} else {
				result = append(result, 0x00)
			}

			totalBits++
		}

		if totalBits == originalLength {
			// Stop unpacking if we've reached the original length.
			break
		}
	}

	return result
}

func BytesToPackets(byteData []byte, NumBitPacket int) [][]byte {
	// Calculate the total number of bits in the byteData
	totalBits := len(byteData) * 8
	// Calculate the number of packets needed based on the NumBitPacket
	numPackets := totalBits / NumBitPacket
	if totalBits%NumBitPacket != 0 {
		numPackets++ // Add an extra packet if there's a remainder
	}

	packets := make([][]byte, numPackets)
	bitIndex := 0 // To keep track of which bit we're on across the byteData

	for i := 0; i < numPackets; i++ {
		packet := make([]byte, NumBitPacket)
		for j := 0; j < NumBitPacket; j++ {
			if bitIndex >= totalBits {
				// If we've processed all bits, remaining packet bits are set to 0
				packet[j] = 0
				continue
			}
			byteIndex := bitIndex / 8                            // Find the byte index in the byteData
			bitPosition := 7 - (bitIndex % 8)                    // Calculate bit position within the current byte
			bitValue := (byteData[byteIndex] >> bitPosition) & 1 // Extract the bit value
			packet[j] = bitValue
			bitIndex++ // Move to the next bit
		}
		packets[i] = packet
	}

	return packets
}

func PacketsToBytes(packets [][]byte, NumBitPacket int, originalPacketSize int) []byte {
	totalBytes := originalPacketSize / 8
	if originalPacketSize%8 != 0 {
		fmt.Print("Wrong originalPacketSize\n")
		return nil
	}

	byteData := make([]byte, totalBytes)
	bitIndex := 0

	for _, packet := range packets {
		for _, bit := range packet {
			if bitIndex >= originalPacketSize {
				break // Stop processing once all original bits are processed
			}
			byteIndex := bitIndex / 8
			bitPosition := 7 - (bitIndex % 8)
			if bit == 1 {
				byteData[byteIndex] |= 1 << bitPosition
			}
			bitIndex++
		}
	}

	return byteData
}
