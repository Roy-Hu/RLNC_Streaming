package matrix

import (
	"runtime"
	"sync"

	"github.com/cloud9-tools/go-galoisfield"
	"github.com/itzmeanjan/kodr"
)

type Matrix [][]byte

// Cell by cell value comparision of two matrices, which
// returns `true` only if all cells are found to be equal
func (m *Matrix) Cmp(m_ Matrix) bool {
	if m.Rows() != m_.Rows() || m.Cols() != m_.Cols() {
		return false
	}

	for i := range *m {
		for j := range (*m)[i] {
			if (*m)[i][j] != m_[i][j] {
				return false
			}
		}
	}
	return true
}

// #-of rows in matrix
//
// This may change in runtime, when some rows are removed
// as they're found to be linearly dependent with some other
// row, after application of RREF
func (m *Matrix) Rows() uint {
	return uint(len(*m))
}

// #-of columns in matrix
//
// This isn't expected to change after initialised
func (m *Matrix) Cols() uint {
	return uint(len((*m)[0]))
}

// Multiplies two matrices ( which can be multiplied )
// in order `m x with`
func (m *Matrix) Multiply(field *galoisfield.GF, with Matrix) (Matrix, error) {
	if m.Cols() != with.Rows() {
		return nil, kodr.ErrMatrixDimensionMismatch
	}

	mult := make(Matrix, m.Rows())
	for i := range mult {
		mult[i] = make([]byte, with.Cols())
	}

	var wg sync.WaitGroup
	tasks := make(chan int, m.Rows()) // Channel for row indices

	// Start a fixed number of worker goroutines
	for w := 0; w < runtime.NumCPU(); w++ {
		go func() {
			for i := range tasks {
				for j := 0; j < int(with.Cols()); j++ {
					for k := 0; k < int(m.Cols()); k++ {
						mult[i][j] = field.Add(mult[i][j], field.Mul((*m)[i][k], with[k][j]))
					}
				}
				wg.Done()
			}
		}()
	}

	// Distribute tasks
	for i := 0; i < int(m.Rows()); i++ {
		wg.Add(1)
		tasks <- i
	}
	close(tasks) // Close channel to terminate workers after tasks are done
	wg.Wait()    // Wait for all tasks to complete

	return mult, nil
}
