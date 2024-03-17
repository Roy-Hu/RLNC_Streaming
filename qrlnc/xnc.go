package qrlnc

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"errors"
	"fmt"
)

// 8192 bits = 1024 bytes
var PktNumBit int = 2048
var RngSeed int64 = int64(1)

var INIT byte = 0x1
var DATA byte = 0x2

// Init Pkt
// Type + ChunkId + FileSize + NumSymbols
// DATA pkt
// Type + ChunkId + Packet

// func EncodeXNCPkt(data XNC) []byte {
// 	pkt := []byte{}

// 	pkt = append(pkt, data.Type)

// 	if data.Type == INIT {
// 		chunkId := make([]byte, 4)
// 		binary.BigEndian.PutUint32(chunkId, uint32(data.ChunkId))
// 		pkt = append(pkt, chunkId...)

// 		filesize := make([]byte, 4)
// 		binary.BigEndian.PutUint32(filesize, uint32(data.FileSize))
// 		pkt = append(pkt, filesize...)

// 		NumSymbols := make([]byte, 4)
// 		binary.BigEndian.PutUint32(NumSymbols, uint32(data.NumSymbols))
// 		pkt = append(pkt, NumSymbols...)
// 	} else {
// 		chunkId := make([]byte, 4)
// 		binary.BigEndian.PutUint32(chunkId, uint32(data.ChunkId))
// 		pkt = append(pkt, chunkId...)
// 	}

// 	return pkt
// }

type XNC struct {
	Type        byte
	ChunkId     int
	FileSize    int
	NumSymbols  int
	Coefficient []uint64
	Packet      []uint64
}

func BytesToUint64s(bytes []byte) ([]uint64, int) {
	// Pad the byte slice with zeros to ensure its length is a multiple of 8.
	for len(bytes)%8 != 0 {
		bytes = append(bytes, 0)
	}

	uint64s := make([]uint64, len(bytes)/8)
	for i := 0; i < len(uint64s); i++ {
		uint64s[i] = binary.BigEndian.Uint64(bytes[i*8 : (i+1)*8])
	}

	return uint64s, len(bytes)
}

func Uint64sToBytes(uint64s []uint64, originalLength int) []byte {
	bytes := make([]byte, len(uint64s)*8)
	for i, val := range uint64s {
		binary.BigEndian.PutUint64(bytes[i*8:], val)
	}

	// Return the slice up to the original length, removing padding.
	return bytes[:originalLength]
}

func EncodeXNCToByte(data XNC) ([]byte, error) {
	var buf bytes.Buffer
	encoder := gob.NewEncoder(&buf)
	if err := encoder.Encode(data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func DecodeByteToXNC(data []byte) (XNC, error) {
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
