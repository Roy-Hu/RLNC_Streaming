package qrlnc

import (
	"math/rand"
	"testing"
	"time"
)

// Assume BinaryCoder struct and necessary methods (NewBinaryCoder, consumePacket, getNewCodedPacket, isFullyDecoded, getNumDecoded) are defined in binarycoder.go

func TestBinaryCoder(t *testing.T) {
	// Parameters
	numSymbols := 4
	numBitPacket := 5
	rngSeed := int64(1)

	// Initialization
	rand.Seed(rngSeed)

	encoder := InitBinaryCoder(numSymbols, numBitPacket, rngSeed)
	decoder := InitBinaryCoder(numSymbols, numBitPacket, rngSeed)

	// Initialize encoder with random bit packets
	for packetID := 0; packetID < encoder.numSymbols; packetID++ {
		packet := make([]int, encoder.numBitPacket)
		randomBits := rand.Uint64()
		for i := range packet {
			packet[i] = int((randomBits >> (uint(encoder.numBitPacket) - 1 - uint(i))) & 1)
		}
		coefficients := make([]int, encoder.numSymbols)
		coefficients[packetID] = 1
		encoder.consumePacket(coefficients, packet)
	}

	t.Log("# Setup complete.")

	// Start
	necessaryMessages := 0
	tic := time.Now()

	for !decoder.isFullyDecoded() {
		coefficient, packet := encoder.getNewCodedPacket()
		decoder.consumePacket(coefficient, packet)
		necessaryMessages++
		t.Logf("## Decode %d out of %d", decoder.getNumDecoded(), decoder.numSymbols)
	}

	t.Logf("\n# Finished !!!")

	if equal(decoder.packetVector, encoder.packetVector) {
		t.Logf("## Successfully decoded all packets at the receiver after %d messages.", necessaryMessages)
		t.Logf("## Whole process took %.2f ms.", time.Since(tic).Seconds()*1000)
		t.Logf("## Decoded packet vectors: %v", decoder.packetVector)
		t.Logf("## Encoder packet vectors: %v", encoder.packetVector)
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
