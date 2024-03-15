package qrlnc

import (
	"bytes"
	"encoding/gob"
)

// 8192 bits = 1024 bytes
var PktNumBit int = 1024
var RngSeed int64 = int64(1)

type XNC struct {
	BitSize     int
	NumSymbols  int
	Coefficient []int
	Packet      []int
}

func EncodePacketDataToByte(data XNC) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodePacketDataToByte(data []byte) (XNC, error) {
	buf := bytes.NewBuffer(data)
	decoder := gob.NewDecoder(buf)
	var decodedData XNC
	if err := decoder.Decode(&decodedData); err != nil {
		return XNC{}, err
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

func PacketsToBytes(packets [][]int, NumBitPacket int, originalBitSize int) []byte {
	totalBytes := originalBitSize / 8
	if originalBitSize%8 != 0 {
		totalBytes++ // Account for leftover bits
	}

	byteData := make([]byte, totalBytes)
	bitIndex := 0

	for _, packet := range packets {
		for _, bit := range packet {
			if bitIndex >= originalBitSize {
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
