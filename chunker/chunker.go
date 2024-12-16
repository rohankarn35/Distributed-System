package chunker

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// represents the single file chunk
type Chunk struct {
	Hash     string
	Size     int
	FilePath string
}

func SplitFileIntoChunks(filepath string, chunkSize int, outputDir string) ([]Chunk, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file : %w", err)
	}
	defer file.Close()

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed  to create output directory, %w", err)
	}
	var chunks []Chunk
	buffer := make([]byte, chunkSize)
	chunkIndex := 0

	for {
		byteRead, err := file.Read(buffer)
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading file %w", err)
		}
		if byteRead == 0 {
			break
		}
		hash := sha256.Sum256(buffer[:byteRead])
		hashString := hex.EncodeToString(hash[:])

		chunkFilePath := fmt.Sprintf("%s/chunk_%d_%s", outputDir, chunkIndex, hashString)
		chunkFile, err := os.Create(chunkFilePath)
		if err != nil {
			return nil, fmt.Errorf("filed to create chunk file %w", err)

		}
		if _, err := chunkFile.Write(buffer[:byteRead]); err != nil {
			chunkFile.Close()
			return nil, fmt.Errorf("failed to write to chunk file: %w", err)
		}
		chunkFile.Close()
		chunks = append(chunks, Chunk{
			Hash:     hashString,
			Size:     byteRead,
			FilePath: chunkFilePath,
		})
		chunkIndex++
	}
	return chunks, nil

}

func PrintChunkFile(chunks []Chunk) {
	fmt.Println("File chunk metadata")
	for _, chunk := range chunks {
		fmt.Printf("Hash %s, Size %d bytes, Path %s\n", chunk.Hash, chunk.Size, chunk.FilePath)
	}
}
