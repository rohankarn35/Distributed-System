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
	Hash string
	Size int
	Data []byte
}

func SplitFileIntoChunks(filepath string, chunkSize int, outputDir string) ([]Chunk, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file : %w", err)
	}
	defer file.Close()

	var chunks []Chunk
	buffer := make([]byte, chunkSize)

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

		chunks = append(chunks, Chunk{
			Hash: hashString,
			Size: byteRead,
			Data: append([]byte{}, buffer[:byteRead]...),
		})
	}
	return chunks, nil

}

func PrintChunkFile(chunks []Chunk) {
	fmt.Println("File chunk metadata")
	for _, chunk := range chunks {
		fmt.Printf("Hash %s, Size %d bytes,\n", chunk.Hash, chunk.Size)
	}
}
