package qrlnc

import (
	"encoding/binary"
	"fmt"
)

// 8192 bits = 1024 bytes
var PKTBITNUM int = 8192
var PKTU64NUM int = PKTBITNUM / 64
var PKTBYTENUM int = PKTBITNUM / 8
var RNGSEED int64 = int64(1)

var TYPE_ACK byte = 0x2
var TYPE_INIT_ENC byte = 0x3
var TYPE_INIT_ORG byte = 0x4
var TYPE_XNC_ENC byte = 0x5
var TYPE_XNC_ORG byte = 0x6

var TYPESIZE int = 1
var IDSIZE int = 4
var FILESIZESIZE int = 4

var CHUNKSIZE int = 1 << 17
var SYMBOLNUM int = (CHUNKSIZE * 8) / PKTBITNUM
var COEFNUM int = SYMBOLNUM / 64
var FRAMESIZE_ENC int = TYPESIZE + IDSIZE + FILESIZESIZE + COEFNUM*8 + PKTU64NUM*8
var FRAMESIZE_ORG int = TYPESIZE + IDSIZE + FILESIZESIZE + PKTBYTENUM
var ACKSIZE int = TYPESIZE + IDSIZE
var INITSIZE int = 128
var INFOSIZE int = TYPESIZE + 4 + 4

var END_CHUNK int = 8192

type XNC struct {
	Type        byte
	ChunkId     int
	ChunkSize   int
	Coefficient []uint64
	PktU64      []uint64
	PktByte     []byte
}

type XNC_ACK struct {
	Type    byte
	ChunkId int
}

type XNC_INIT struct {
	Type     byte
	Len      int
	Filename string
}

type XNC_INFO struct {
	Type     byte
	ChunkNum int
	FileSize int
}

func EncodeInfo(data XNC_INFO) ([]byte, error) {
	pkt := []byte{}

	pkt = append(pkt, data.Type)

	chunkNum := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkNum, uint32(data.ChunkNum))
	pkt = append(pkt, chunkNum...)

	fileSize := make([]byte, 4)
	binary.BigEndian.PutUint32(fileSize, uint32(data.FileSize))
	pkt = append(pkt, fileSize...)

	return pkt, nil
}

func EncodeInit(data XNC_INIT) ([]byte, error) {
	pkt := make([]byte, 128)

	pkt[0] = data.Type
	binary.BigEndian.PutUint32(pkt[1:5], uint32(data.Len))

	for i := 0; i < data.Len; i++ {
		pkt[5+i] = data.Filename[i]
	}

	return pkt, nil
}

func DecodeInit(data []byte) (XNC_INIT, error) {
	if data[0] != TYPE_INIT_ENC && data[0] != TYPE_INIT_ORG {
		return XNC_INIT{}, fmt.Errorf("pkt type is not correct")
	}

	init := XNC_INIT{}
	init.Type = data[0]
	init.Len = int(binary.BigEndian.Uint32(data[1:5]))
	init.Filename = string(data[5 : 5+init.Len])

	return init, nil
}

// XNC pkt
// Type + ChunkId + FileSize+ Packet
func EncodeAck(data XNC_ACK) ([]byte, error) {
	pkt := []byte{}

	pkt = append(pkt, data.Type)

	chunkId := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkId, uint32(data.ChunkId))
	pkt = append(pkt, chunkId...)

	return pkt, nil
}

func DecodeAck(data []byte) (XNC_ACK, error) {
	if len(data) != ACKSIZE {
		return XNC_ACK{}, fmt.Errorf("XNC pkt size is not correct")
	}

	if data[0] != TYPE_ACK {
		return XNC_ACK{}, fmt.Errorf("pkt type is not correct")
	}

	ack := XNC_ACK{}

	ack.Type = data[0]
	ack.ChunkId = int(binary.BigEndian.Uint32(data[1:5]))

	return ack, nil
}

func EncodeXNCPkt(data XNC) ([]byte, error) {
	if data.Type == TYPE_XNC_ENC {
		if len(data.Coefficient) != COEFNUM || len(data.PktU64) != PKTU64NUM {
			return nil, fmt.Errorf("XNC pktu64 size is not correct")
		}
	} else if data.Type == TYPE_XNC_ORG {
		if len(data.PktByte) != PKTBYTENUM {
			return nil, fmt.Errorf("XNC pkt byte size is not correct")
		}
	} else {
		return nil, fmt.Errorf("Unknow XNC Type")
	}

	pkt := []byte{}

	pkt = append(pkt, data.Type)

	chunkId := make([]byte, 4)
	binary.BigEndian.PutUint32(chunkId, uint32(data.ChunkId))
	pkt = append(pkt, chunkId...)

	pktsize := make([]byte, 4)
	binary.BigEndian.PutUint32(pktsize, uint32(data.ChunkSize))
	pkt = append(pkt, pktsize...)

	if data.Type == TYPE_XNC_ENC {
		for i := 0; i < COEFNUM; i++ {
			coef := make([]byte, 8)
			binary.BigEndian.PutUint64(coef, uint64(data.Coefficient[i]))
			pkt = append(pkt, coef...)
		}

		for i := 0; i < PKTU64NUM; i++ {
			packet := make([]byte, 8)
			binary.BigEndian.PutUint64(packet, uint64(data.PktU64[i]))
			pkt = append(pkt, packet...)
		}
	} else if data.Type == TYPE_XNC_ORG {
		pkt = append(pkt, data.PktByte...)
	}

	return pkt, nil
}

func DecodeXNCPkt(data []byte) (XNC, error) {
	if len(data) != FRAMESIZE_ENC && len(data) != FRAMESIZE_ORG {
		return XNC{}, fmt.Errorf("XNC pkt size is not correct")
	}

	xnc := XNC{}
	xnc.Type = data[0]
	xnc.ChunkId = int(binary.BigEndian.Uint32(data[1:5]))
	xnc.ChunkSize = int(binary.BigEndian.Uint32(data[5:9]))

	if xnc.Type == TYPE_XNC_ENC {
		for i := 0; i < COEFNUM; i++ {
			xnc.Coefficient = append(xnc.Coefficient, binary.BigEndian.Uint64(data[9+i*8:17+i*8]))
		}

		for i := 0; i < PKTU64NUM; i++ {
			xnc.PktU64 = append(xnc.PktU64, binary.BigEndian.Uint64(data[9+COEFNUM*8+i*8:17+COEFNUM*8+i*8]))
		}
	} else if xnc.Type == TYPE_XNC_ORG {
		xnc.PktByte = data[9:]
	} else {
		return XNC{}, fmt.Errorf("Unknow XNC type")
	}

	return xnc, nil
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