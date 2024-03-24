package qrlnc

import (
	"crypto/tls"
	"fmt"
	"runtime"
	"sync"
)

// binMatRref performs the Row Reduction to Echelon Form on a binary matrix
func binMatRref(A *[][]byte) ([][]byte, int, []bool) {
	B := [][]byte{}
	n := len((*A)[0])

	// Forward sweep
	for col := 0; col < n; col++ {
		numCols := len((*A))
		j := 0
		rows := []int{}
		// Precompute relevant rows
		for j < numCols {
			if (*A)[j][col] == 1 {
				rows = append(rows, j)
			}
			j++
		}

		// Process each row
		if len(rows) >= 1 {
			for c := 1; c < len(rows); c++ {
				for k := 0; k < n; k++ {
					(*A)[rows[c]][k] = ((*A)[rows[c]][k] + (*A)[rows[0]][k]) % 2
				}
			}
			B = append(B, (*A)[rows[0]]) // Copy for backwards sweep
			// Remove the processed row
			*A = append((*A)[:rows[0]], (*A)[rows[0]+1:]...)
		}
	}

	n = len(B)
	nk := len(B[0])
	// Backwards sweep
	for row := n - 1; row >= 0; row-- {
		// Find leading one
		leadingOne := -1
		for i, val := range B[row][:n] {
			if val == 1 {
				leadingOne = i
				break
			}
		}

		if leadingOne != -1 {
			for toReduceRow := row - 1; toReduceRow >= 0; toReduceRow-- {
				if B[toReduceRow][leadingOne] == 1 {
					for k := 0; k < nk; k++ {
						B[toReduceRow][k] = (B[toReduceRow][k] + B[row][k]) % 2
					}
				}
			}
		}
	}

	symbolCutoff := len(B[0]) / 2
	rowSums := make([]int, len(B))
	rank := 0
	isDecoded := make([]bool, symbolCutoff)

	for i, row := range B {
		leadingOne := false

		for j, val := range row {
			if j >= symbolCutoff {
				break
			}
			rowSums[i] += int(val)
			if val == 1 && !leadingOne {
				leadingOne = true
				rank++
			}
		}

		if rowSums[i] == 1 && i < symbolCutoff {
			isDecoded[i] = true
		}
	}

	return B, rank, isDecoded
}

// binMatDot performs dot product of two binary matrices
func binMatDot(K, L [][]byte) [][]byte {
	numRows := len(K)
	numBits := len(L[0])
	result := make([][]byte, numRows)

	var wg sync.WaitGroup
	numGoroutines := runtime.NumCPU() // For example, could be runtime.NumCPU() for dynamic allocation
	rowsPerGoroutine := (numRows + numGoroutines - 1) / numGoroutines

	for i := 0; i < numGoroutines; i++ {
		start := i * rowsPerGoroutine
		end := start + rowsPerGoroutine
		if end > numRows {
			end = numRows
		}
		wg.Add(1)
		go func(start, end int) {
			defer wg.Done()
			for row := start; row < end; row++ {
				rowSolution := make([]byte, numBits)
				for k := range K[row] {
					if K[row][k] != 0 {
						for j := range L[k] {
							rowSolution[j] ^= (K[row][k] & L[k][j])
						}
					}
				}
				result[row] = rowSolution
			}
		}(start, end)
	}

	wg.Wait()
	return result
}

// identity generates an identity matrix of size n
func identity(n int) [][]byte {
	id := make([][]byte, n)
	for i := range id {
		id[i] = make([]byte, n)
		id[i][i] = 1
	}
	return id
}

// Helper function to sum a slice of integers
func rowSum(slice []byte) int {
	total := 0
	for _, val := range slice {
		total += int(val)
	}
	return total
}

func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func GenerateTLSConfig() *tls.Config {
	cert, err := tls.LoadX509KeyPair("../godash/http/certs/cert.pem", "../godash/http/certs/key.pem")
	if err != nil {
		fmt.Printf("TLS config err: %v", err)

		return nil
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
	}
}

// compressBinaryMatrix compresses a binary matrix represented as [][]byte into [][]uint64.
// Each uint64 value will represent up to 64 binary values from the original matrix.
func compressBinaryMatrix(matrix [][]byte) [][]uint64 {
	var result [][]uint64

	// Determine the size of the uint64 row required to represent each row of the matrix.
	var uint64RowSize int
	if len(matrix) > 0 {
		uint64RowSize = (len(matrix[0]) + 63) / 64 // Ceiling division to accommodate all bits.
	}

	for _, row := range matrix {
		uint64Row := make([]uint64, uint64RowSize)
		for i, bit := range row {
			if bit == 1 {
				// Calculate which uint64 value and bit position to set.
				uint64Index := i / 64                      // Determine which uint64 element.
				bitPosition := uint(i % 64)                // Determine the bit position within that uint64.
				uint64Row[uint64Index] |= 1 << bitPosition // Set the bit using a bit operation.
			}
		}
		result = append(result, uint64Row)
	}

	return result
}

// decompressBinaryMatrix decompresses a binary matrix represented as [][]uint64 back into [][]byte.
func decompressBinaryMatrix(matrix [][]uint64, originalRowLength int) [][]byte {
	var result [][]byte

	for _, uint64Row := range matrix {
		byteRow := make([]byte, originalRowLength)
		for i := range byteRow {
			uint64Index := i / 64       // Determine which uint64 element to check.
			bitPosition := uint(i % 64) // Determine the bit position within that uint64.
			if uint64Index < len(uint64Row) && (uint64Row[uint64Index]&(1<<bitPosition)) != 0 {
				byteRow[i] = 1
			} else {
				byteRow[i] = 0
			}
		}
		result = append(result, byteRow)
	}

	return result
}
