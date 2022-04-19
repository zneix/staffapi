package main

import (
	"errors"
	"fmt"
)

// NewErrorf formats according to a format specifier and runs errors.New with the resulting string
func NewErrorf(format string, a ...any) error {
	return errors.New(fmt.Sprintf(format, a...))
}

// ChunkStringSlice takes 1 big slice of strings and splits it into
// smaller chunks with the maximum amount of strings specified by chunkSize
func ChunkStringSlice(slice []string, chunkSize int) (chunks [][]string) {
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize

		// necessary check to avoid slicing beyond slice capacity
		if end > len(slice) {
			end = len(slice)
		}

		chunks = append(chunks, slice[i:end])
	}

	return
}
