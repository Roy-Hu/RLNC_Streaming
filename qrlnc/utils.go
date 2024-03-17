package qrlnc

import (
	"crypto/tls"
	"fmt"
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
	rowSums := make([]byte, len(B))
	for i, row := range B {
		for _, val := range row[:symbolCutoff] {
			rowSums[i] += val
		}
	}
	rank := 0
	for _, rSum := range rowSums {
		if rSum >= 1 {
			rank++
		}
	}
	decodedSymbols := make(map[int]bool)
	for _, row := range B {
		if rowSum(row[:symbolCutoff]) == 1 {
			for i, val := range row[:symbolCutoff] {
				if val == 1 {
					decodedSymbols[i] = true
					break
				}
			}
		}
	}
	isDecoded := make([]bool, n)
	for i := range isDecoded {
		_, found := decodedSymbols[i]
		isDecoded[i] = found
	}
	return B, rank, isDecoded
}

// binMatDot performs dot product of two binary matrices
func binMatDot(K [][]byte, L [][]byte) [][]byte {
	numRows := len(K)
	numBits := len(L[0])
	result := make([][]byte, numRows)

	var wg sync.WaitGroup
	wg.Add(numRows) // Set the number of goroutines to wait for

	for row := range K {
		go func(row int) {
			defer wg.Done() // Signal that this goroutine is done
			rowSolution := make([]byte, numBits)
			for k := range K[row] {
				if K[row][k] != 0 {
					for j := range L[k] {
						rowSolution[j] = (rowSolution[j] + K[row][k]*L[k][j]) & 1
					}
				}
			}
			result[row] = rowSolution
		}(row)
	}

	wg.Wait() // Wait for all goroutines to finish
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
