package main

import (
	"distributed-cas/service"
	"fmt"
	"log"
	"path/filepath"
)

func main() {
	// Configuration
	basePath := "./storage"
	chunkSize := 1024
	numNodes := 5

	// Create and initialize storage service
	service := service.NewStorageService(basePath, chunkSize, numNodes)
	if err := service.Initialize(); err != nil {
		log.Fatal("Failed to initialize storage:", err)
	}

	// // Example usage with command line argument
	// if len(os.Args) < 2 {
	// 	log.Fatal("Please provide a file path as argument")
	// }

	filePath := "/home/rohankarn487/Documents/name.pdf"
	fmt.Printf("Processing file: %s\n", filePath)

	// Store file
	metadata, err := service.StoreFile(filePath)
	if err != nil {
		log.Fatal("Failed to store file:", err)
	}

	// Print node statistics
	stats := service.GetNodeStatistics()
	fmt.Printf("\nNode Statistics:\n")
	for _, node := range stats {
		fmt.Printf("Node %d: %d bytes used, %d chunks\n",
			node.NodeID, node.UsedSpace, node.NumChunks)
	}

	// Reconstruct file
	outputPath := filepath.Join(".", "reconstructed"+metadata.Type)
	if err := service.ReconstructFile(metadata, outputPath); err != nil {
		log.Fatal("Failed to reconstruct file:", err)
	}

	fmt.Printf("\nFile processed successfully:\n")
	fmt.Printf("Original size: %d bytes\n", metadata.Size)
	fmt.Printf("Number of chunks: %d\n", len(metadata.Chunks))
	fmt.Printf("Reconstructed file: %s\n", outputPath)
}
