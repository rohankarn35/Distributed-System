package main

import (
	"distributed-cas/chunker"
	"fmt"
	"log"
)

func main() {
	filepath := "/home/rohankarn487/Documents/test/rohan.mp4"
	chunkSize := 1024 * 1024
	outputDir := "/home/rohankarn487/Documents/test"

	chunks, err := chunker.SplitFileIntoChunks(filepath, chunkSize, outputDir)
	if err != nil {
		log.Fatal("error splitting file %w", err)

	}
	chunker.PrintChunkFile(chunks)
	fmt.Println("file has been successfully split into chunks!")
}
