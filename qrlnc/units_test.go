package qrlnc // Change this to the name of your package

import (
	"reflect"
	"testing"
)

func TestBinMatRref_SecondComplexMatrix(t *testing.T) {
	matrix := &[][]byte{
		{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 1, 0, 1, 0, 0, 0, 1, 0, 0, 0},
		{0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0},
		{1, 1, 1, 1, 0, 0, 0, 0, 0, 0, 0, 1},
	}
	expectedMatrix := [][]byte{
		{1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 0},
		{0, 1, 0, 0, 0, 0, 0, 1, 0, 0, 0, 0},
		{0, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 1},
		{0, 0, 0, 1, 0, 0, 0, 0, 0, 1, 0, 0},
		{0, 0, 0, 0, 1, 0, 1, 1, 1, 1, 0, 1},
		{0, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0},
	}
	expectedRank := 6
	expectedDecoded := []bool{true, true, true, true, true, true}

	matrixResult, rankResult, decodedResult := binMatRref(matrix)
	if !reflect.DeepEqual(matrixResult, expectedMatrix) {
		t.Errorf("Matrix Result mismatch. Got %v, expected %v", matrixResult, expectedMatrix)
	}
	if rankResult != expectedRank {
		t.Errorf("Rank Result mismatch. Got %v, expected %v", rankResult, expectedRank)
	}
	if !reflect.DeepEqual(decodedResult, expectedDecoded) {
		t.Errorf("Decoded Result mismatch. Got %v, expected %v", decodedResult, expectedDecoded)
	}
}
