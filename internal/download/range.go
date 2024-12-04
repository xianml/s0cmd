package download

import (
	"fmt"
	"math"
)

const (
	// BlockSize64MB represents 64MB in bytes
	BlockSize64MB = int64(64 * 1024 * 1024)
)

// CalculateRange calculates the start and end byte ranges for parallel downloads.
// Each range starts at 64MB boundaries and ensures no overlaps between ranges.
func CalculateRange(objectSize int64, parallelism int) ([][2]int64, error) {
	if parallelism <= 0 {
		return nil, fmt.Errorf("parallelism must be greater than 0")
	}

	if objectSize <= 0 {
		return nil, fmt.Errorf("object size must be greater than 0")
	}

	// Calculate approximate part size rounded up to 64MB blocks
	basePartSize := int64(math.Ceil(float64(objectSize) / float64(parallelism)))
	partSize := int64(math.Ceil(float64(basePartSize)/float64(BlockSize64MB))) * BlockSize64MB

	// Recalculate parallelism if part size became too large
	actualParallelism := int(math.Ceil(float64(objectSize) / float64(partSize)))
	if actualParallelism < parallelism {
		parallelism = actualParallelism
	}

	ranges := make([][2]int64, parallelism)
	var start, end int64

	for i := 0; i < parallelism; i++ {
		start = int64(i) * partSize
		if i == parallelism-1 {
			// Last range should extend to the end of file
			end = objectSize - 1
		} else {
			end = start + partSize - 1
			if end >= objectSize {
				end = objectSize - 1
			}
		}
		ranges[i] = [2]int64{start, end}
	}

	// Validate that ranges cover entire object
	if ranges[len(ranges)-1][1] != objectSize-1 {
		return nil, fmt.Errorf("calculated ranges do not cover entire object size")
	}

	return ranges, nil
}
