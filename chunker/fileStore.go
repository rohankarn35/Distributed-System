package chunker

import (
	"fmt"
	"os"
	"path/filepath"
)

func StoreChunks(chunks []Chunk, outputDir string) (map[int]string, error) {
	chunkPaths := make(map[int]string)

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create output directory %w", err)
	}
	for index, chunk := range chunks {
		chunkFileName := fmt.Sprintf("chunk_%d_%s", index, chunk.Hash)
		chunkFilePath := filepath.Join(outputDir, chunkFileName)
		chunkFile, err := os.Create(chunkFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to create chunks %w", err)
		}
		_, err = chunkFile.Write(chunk.Data)
		if err != nil {
			chunkFile.Close()
			return nil, fmt.Errorf("failed to write to chunk file %w", err)
		}
		chunkFile.Close()
		chunkPaths[index] = chunkFilePath
	}
	return chunkPaths, nil
}
