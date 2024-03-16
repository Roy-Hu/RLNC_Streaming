package qrlnc

import (
	"fmt"
	"math/rand"
)

type BinaryCoder struct {
	NumSymbols        int
	NumBitPacket      int
	rng               *rand.Rand
	numIndependent    int
	symbolDecoded     []bool
	id                [][]byte
	coefficientMatrix [][]byte
	PacketVector      [][]int
}

func InitBinaryCoder(NumSymbols int, packetSize int, RngSeed int64) *BinaryCoder {
	rng := rand.New(rand.NewSource(RngSeed))
	bc := &BinaryCoder{
		NumSymbols:        NumSymbols,
		NumBitPacket:      packetSize,
		rng:               rng,
		numIndependent:    0,
		symbolDecoded:     make([]bool, NumSymbols),
		id:                identity(NumSymbols),
		coefficientMatrix: make([][]byte, NumSymbols),
		PacketVector:      make([][]int, NumSymbols),
	}
	bc.reset()
	return bc
}

func (bc *BinaryCoder) reset() {
	bc.numIndependent = 0
	bc.symbolDecoded = make([]bool, bc.NumSymbols)
	bc.id = identity(bc.NumSymbols)

	// python: self.coefficient_matrix = [ [0] * self.num_symbols + self.id[k] for k in range(self.num_symbols)] # save current rref to reduce computational load in the future
	for k := 0; k < bc.NumSymbols; k++ {
		bc.coefficientMatrix[k] = make([]byte, 2*bc.NumSymbols)
		for j := 0; j < bc.NumSymbols; j++ {
			bc.coefficientMatrix[k][bc.NumSymbols+j] = bc.id[k][j]
		}
	}
	bc.PacketVector = make([][]int, bc.NumSymbols)
}

func (bc *BinaryCoder) isSymbolDecoded(index int) bool {
	if index < 0 || index >= bc.NumSymbols {
		return false
	}
	return bc.symbolDecoded[index]
}

func (bc *BinaryCoder) getDecodedSymbol(index int) []int {
	if bc.isSymbolDecoded(index) {
		return bc.PacketVector[index]
	}

	return nil
}

func (bc *BinaryCoder) GetNumDecoded() int {
	sum := 0

	for s := range bc.symbolDecoded {
		if bc.symbolDecoded[s] {
			sum++
		}
	}

	return sum
}

func (bc *BinaryCoder) IsFullyDecoded() bool {
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

func (bc *BinaryCoder) ConsumePacket(coefficients []byte, packet []int) {
	if !bc.IsFullyDecoded() {
		copy(bc.coefficientMatrix[bc.numIndependent], coefficients)

		bc.PacketVector[bc.numIndependent] = packet

		var extendedRref [][]byte

		extendedRref, bc.numIndependent, bc.symbolDecoded = binMatRref(&bc.coefficientMatrix)

		transformation := make([][]byte, len(extendedRref))
		for i, row := range extendedRref {
			transformation[i] = make([]byte, bc.NumSymbols)
			copy(transformation[i], row[bc.NumSymbols:2*bc.NumSymbols])
		}

		bc.PacketVector = binMatDot(transformation, bc.PacketVector)

		rref := make([][]byte, len(extendedRref))
		for i, row := range extendedRref {
			rref[i] = make([]byte, bc.NumSymbols)
			copy(rref[i], row[:bc.NumSymbols])
		}

		bc.coefficientMatrix = make([][]byte, bc.NumSymbols)
		for k := 0; k < bc.NumSymbols; k++ {
			bc.coefficientMatrix[k] = append(rref[k], bc.id[k]...)
		}
	}
}

func (bc *BinaryCoder) getSysCodedPacket(index int) ([]int, []int) {
	if index < 0 || index >= bc.NumSymbols {
		return nil, nil
	}

	if bc.isSymbolDecoded(index) {
		coefficients := make([]int, bc.NumSymbols)
		coefficients[index] = 1
		return coefficients, bc.PacketVector[index]
	}

	return nil, nil
}

func (bc *BinaryCoder) GetNewCodedPacket() ([]byte, []int) {
	coefficients := make([]byte, bc.NumSymbols)
	packet := make([]int, bc.NumBitPacket)

	var randomDecisions []int

	for rowSum(coefficients) == 0 {
		randomNum := bc.rng.Intn(bc.numIndependent) + 1
		randomDecisions = make([]int, randomNum)
		for i := range randomDecisions {
			randomDecisions[i] = bc.rng.Intn(bc.numIndependent)
		}

		coefficients = make([]byte, bc.NumSymbols)
		for i := range coefficients {
			for _, selected := range randomDecisions {
				coefficients[i] ^= bc.coefficientMatrix[selected][i]
			}
		}
	}

	for i := range packet {
		for _, selected := range randomDecisions {
			packet[i] ^= bc.PacketVector[selected][i]
		}
	}

	return coefficients, packet
}

func (bc *BinaryCoder) GetNewCodedPacketByte(fileSize int) ([]byte, error) {
	coefficient, packet := bc.GetNewCodedPacket()

	coef := bytesToUint64s(coefficient)
	xncPkt := XNC{
		FileSize:    fileSize,
		NumSymbols:  bc.NumSymbols,
		Coefficient: coef,
		Packet:      packet,
	}

	encodedPkt, err := EncodePacketDataToByte(xncPkt)

	if err != nil {
		fmt.Println("Error encoding packet data:", err)
		return nil, err
	}

	return encodedPkt, nil
}
