package qrlnc

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
)

// 8192 bits = 1024 bytes
var PktNumBit int = 2048
var RngSeed int64 = int64(1)

type XNC struct {
	FileSize    int
	NumSymbols  int
	Coefficient []uint64
	Packet      []int
}

func bytesToUint64s(bytes []byte) []uint64 {
	// Pad the byte slice with zeros to ensure its length is a multiple of 8.
	for len(bytes)%8 != 0 {
		bytes = append(bytes, 0)
	}

	uint64s := make([]uint64, len(bytes)/8)
	for i := 0; i < len(uint64s); i++ {
		uint64s[i] = binary.BigEndian.Uint64(bytes[i*8 : (i+1)*8])
	}

	return uint64s
}

func uint64sToBytes(uint64s []uint64, originalLength int) []byte {
	bytes := make([]byte, len(uint64s)*8)
	for i, val := range uint64s {
		binary.BigEndian.PutUint64(bytes[i*8:], val)
	}

	// Return the slice up to the original length, removing padding.
	return bytes[:originalLength]
}

func EncodePacketDataToByte(data XNC) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeByteToPacketData(data []byte) (XNC, error) {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	var decodedData XNC
	if err := decoder.Decode(&decodedData); err != nil {
		return XNC{}, err
	}
	if buf.Len() > 0 {
		return XNC{}, errors.New("data contains more bytes than expected")
	}
	return decodedData, nil
}

func BytesToPackets(byteData []byte, NumBitPacket int) [][]int {
	// Calculate the total number of bits in the byteData
	totalBits := len(byteData) * 8
	// Calculate the number of packets needed based on the NumBitPacket
	numPackets := totalBits / NumBitPacket
	if totalBits%NumBitPacket != 0 {
		numPackets++ // Add an extra packet if there's a remainder
	}

	packets := make([][]int, numPackets)
	bitIndex := 0 // To keep track of which bit we're on across the byteData

	for i := 0; i < numPackets; i++ {
		packet := make([]int, NumBitPacket)
		for j := 0; j < NumBitPacket; j++ {
			if bitIndex >= totalBits {
				// If we've processed all bits, remaining packet bits are set to 0
				packet[j] = 0
				continue
			}
			byteIndex := bitIndex / 8                            // Find the byte index in the byteData
			bitPosition := 7 - (bitIndex % 8)                    // Calculate bit position within the current byte
			bitValue := (byteData[byteIndex] >> bitPosition) & 1 // Extract the bit value
			packet[j] = int(bitValue)
			bitIndex++ // Move to the next bit
		}
		packets[i] = packet
	}

	return packets
}

func PacketsToBytes(packets [][]int, NumBitPacket int, originalPacketSize int) []byte {
	totalBytes := originalPacketSize / 8
	if originalPacketSize%8 != 0 {
		totalBytes++ // Account for leftover bits
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
