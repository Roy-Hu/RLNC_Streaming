package qrlnc

import (
	"math/rand"
	"testing"
	"time"
)

// Assume BinaryCoder struct and necessary methods (NewBinaryCoder, ConsumePacket, GetNewCodedPacket, IsFullyDecoded, GetNumDecoded) are defined in binarycoder.go

func TestBinaryCoder(t *testing.T) {
	// Parameters
	NumSymbols := SYMBOLNUM
	NumBitPacket := PKTBITNUM
	RNGSEED := int64(1)

	// Initialization
	rand.Seed(RNGSEED)

	encoder := InitBinaryCoder(NumSymbols, NumBitPacket, RNGSEED)
	decoder := InitBinaryCoder(NumSymbols, NumBitPacket, RNGSEED)

	packets := make([][]byte, NumSymbols)
	// Initialize encoder with random bit packets
	for packetID := 0; packetID < encoder.NumSymbols; packetID++ {
		packet := make([]byte, encoder.NumBitPacket)
		randomBits := rand.Uint64()
		for i := range packet {
			packet[i] = byte((randomBits >> (uint(encoder.NumBitPacket) - 1 - uint(i))) & 1)
		}

		packets[packetID] = packet
		coefficients := make([]byte, encoder.NumSymbols)
		coefficients[packetID] = 1
		encoder.ConsumePacket(coefficients, packet)
	}

	t.Log("# Setup complete.")

	// Start
	necessaryMessages := 0
	tic := time.Now()

	for !decoder.IsFullyDecoded() {
		coefficient, packet := encoder.GetNewCodedPacket()
		decoder.ConsumePacket(coefficient, packet)
		necessaryMessages++
		t.Logf("## Get %d, Decode %d out of %d", necessaryMessages, decoder.GetNumDecoded(), decoder.NumSymbols)
	}

	t.Logf("\n# Finished !!!")

	if equal(decoder.PacketVector, packets) {
		t.Logf("## Successfully decoded all packets at the receiver after %d messages.", necessaryMessages)
		t.Logf("## Whole process took %.2f ms.", time.Since(tic).Seconds()*1000)
	} else {
		t.Error("## Error, decoded packet vectors are not equal!!!")
		t.Errorf("## Encoder: %v", packets[NumSymbols-1])
		t.Errorf("## Decoder: %v", decoder.PacketVector[NumSymbols-1])
	}
}

// Helper function to check if two 2D slices are equal
func equal(a, b [][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := 0; j < len(a[0]); j++ {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
