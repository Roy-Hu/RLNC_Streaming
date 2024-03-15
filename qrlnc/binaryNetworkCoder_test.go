package qrlnc

import (
	"math/rand"
	"testing"
	"time"
)

// Assume BinaryCoder struct and necessary methods (NewBinaryCoder, ConsumePacket, GetNewCodedPacket, IsFullyDecoded, GetNumDecoded) are defined in binarycoder.go

func TestBinaryCoder(t *testing.T) {
	// Parameters
	NumSymbols := 4
	NumBitPacket := 5
	RngSeed := int64(1)

	// Initialization
	rand.Seed(RngSeed)

	encoder := InitBinaryCoder(NumSymbols, NumBitPacket, RngSeed)
	decoder := InitBinaryCoder(NumSymbols, NumBitPacket, RngSeed)

	// Initialize encoder with random bit packets
	for packetID := 0; packetID < encoder.NumSymbols; packetID++ {
		packet := make([]int, encoder.NumBitPacket)
		randomBits := rand.Uint64()
		for i := range packet {
			packet[i] = int((randomBits >> (uint(encoder.NumBitPacket) - 1 - uint(i))) & 1)
		}
		coefficients := make([]int, encoder.NumSymbols)
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
		t.Logf("## Decode %d out of %d", decoder.GetNumDecoded(), decoder.NumSymbols)
	}

	t.Logf("\n# Finished !!!")

	if equal(decoder.PacketVector, encoder.PacketVector) {
		t.Logf("## Successfully decoded all packets at the receiver after %d messages.", necessaryMessages)
		t.Logf("## Whole process took %.2f ms.", time.Since(tic).Seconds()*1000)
		t.Logf("## Decoded packet vectors: %v", decoder.PacketVector)
		t.Logf("## Encoder packet vectors: %v", encoder.PacketVector)
	} else {
		t.Error("## Error, decoded packet vectors are not equal!!!")
	}
}

// Helper function to check if two 2D slices are equal
func equal(a, b [][]int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if len(a[i]) != len(b[i]) {
			return false
		}
		for j := range a[i] {
			if a[i][j] != b[i][j] {
				return false
			}
		}
	}
	return true
}
