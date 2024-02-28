package qrlnc

import (
	"math/rand"
)

type BinaryCoder struct {
	numSymbols        int
	numBitPacket      int
	rng               *rand.Rand
	numIndependent    int
	symbolDecoded     []bool
	id                [][]int
	coefficientMatrix [][]int
	packetVector      [][]int
}

func InitBinaryCoder(numSymbols int, packetSize int, rngSeed int64) *BinaryCoder {
	rng := rand.New(rand.NewSource(rngSeed))
	bc := &BinaryCoder{
		numSymbols:        numSymbols,
		numBitPacket:      packetSize,
		rng:               rng,
		numIndependent:    0,
		symbolDecoded:     make([]bool, numSymbols),
		id:                identity(numSymbols),
		coefficientMatrix: make([][]int, numSymbols),
		packetVector:      make([][]int, numSymbols),
	}
	bc.reset()
	return bc
}

func (bc *BinaryCoder) reset() {
	bc.numIndependent = 0
	bc.symbolDecoded = make([]bool, bc.numSymbols)
	bc.id = identity(bc.numSymbols)

	// python: self.coefficient_matrix = [ [0] * self.num_symbols + self.id[k] for k in range(self.num_symbols)] # save current rref to reduce computational load in the future
	for k := 0; k < bc.numSymbols; k++ {
		bc.coefficientMatrix[k] = make([]int, 2*bc.numSymbols)
		for j := 0; j < bc.numSymbols; j++ {
			bc.coefficientMatrix[k][bc.numSymbols+j] = bc.id[k][j]
		}
	}
	bc.packetVector = make([][]int, bc.numSymbols)
}

func (bc *BinaryCoder) isSymbolDecoded(index int) bool {
	if index < 0 || index >= bc.numSymbols {
		return false
	}
	return bc.symbolDecoded[index]
}

func (bc *BinaryCoder) getDecodedSymbol(index int) []int {
	if bc.isSymbolDecoded(index) {
		return bc.packetVector[index]
	}

	return nil
}

func (bc *BinaryCoder) getNumDecoded() int {
	sum := 0

	for s := range bc.symbolDecoded {
		if bc.symbolDecoded[s] {
			sum++
		}
	}

	return sum
}

func (bc *BinaryCoder) isFullyDecoded() bool {
	for _, decoded := range bc.symbolDecoded {
		if !decoded {
			return false
		}
	}
	return true
}

func (bc *BinaryCoder) rank() int {
	return bc.numIndependent
}

func (bc *BinaryCoder) consumePacket(coefficients []int, packet []int) {
	if !bc.isFullyDecoded() {
		copy(bc.coefficientMatrix[bc.numIndependent], coefficients)

		bc.packetVector[bc.numIndependent] = packet

		var extendedRref [][]int

		extendedRref, bc.numIndependent, bc.symbolDecoded = binMatRref(&bc.coefficientMatrix)

		transformation := make([][]int, len(extendedRref))
		for i, row := range extendedRref {
			transformation[i] = make([]int, bc.numSymbols)
			copy(transformation[i], row[bc.numSymbols:2*bc.numSymbols])
		}

		bc.packetVector = binMatDot(transformation, bc.packetVector)

		rref := make([][]int, len(extendedRref))
		for i, row := range extendedRref {
			rref[i] = make([]int, bc.numSymbols)
			copy(rref[i], row[:bc.numSymbols])
		}

		bc.coefficientMatrix = make([][]int, bc.numSymbols)
		for k := 0; k < bc.numSymbols; k++ {
			bc.coefficientMatrix[k] = append(rref[k], bc.id[k]...)
		}
	}
}

func (bc *BinaryCoder) getSysCodedPacket(index int) ([]int, []int) {
	if index < 0 || index >= bc.numSymbols {
		return nil, nil
	}

	if bc.isSymbolDecoded(index) {
		coefficients := make([]int, bc.numSymbols)
		coefficients[index] = 1
		return coefficients, bc.packetVector[index]
	}

	return nil, nil
}

func (bc *BinaryCoder) getNewCodedPacket() ([]int, []int) {
	coefficients := make([]int, bc.numSymbols)
	packet := make([]int, bc.numBitPacket)

	var randomDecisions []int

	for rowSum(coefficients) == 0 {
		randomNum := bc.rng.Intn(bc.numIndependent) + 1
		randomDecisions = make([]int, randomNum)
		for i := range randomDecisions {
			randomDecisions[i] = bc.rng.Intn(bc.numIndependent)
		}

		coefficients = make([]int, bc.numSymbols)
		for i := range coefficients {
			for _, selected := range randomDecisions {
				coefficients[i] ^= bc.coefficientMatrix[selected][i]
			}
		}
	}

	for i := range packet {
		for _, selected := range randomDecisions {
			packet[i] ^= bc.packetVector[selected][i]
		}
	}

	return coefficients, packet
}
