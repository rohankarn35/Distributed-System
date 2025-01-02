package service

import (
	"crypto/sha256"
	"distributed-cas/domain"
	"distributed-cas/models"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type storageService struct {
	chunkSize int
	numNodes  int
	basePath  string
	mu        sync.RWMutex
	nodes     []models.NodeStatus
}

// ChunkWorker implements domain.StorageService.
func (s *storageService) ChunkWorker(wg *sync.WaitGroup, chunks <-chan []byte, errorChan chan<- error, metadata *models.FileMetadata) {
	for chunk := range chunks {
		hash := s.HashData(chunk)
		chunkSize := int64(len(chunk))

		nodeNum := s.SelectOptimalNode(chunkSize)

		s.mu.Lock()
		chunkInfo := models.ChunkInfo{
			H: hash[:16],
			I: uint32(len(metadata.Chunks)),
			N: nodeNum,
			S: chunkSize,
		}
		metadata.Chunks = append(metadata.Chunks, chunkInfo)

		s.nodes[nodeNum].UsedSpace += chunkSize
		s.nodes[nodeNum].NumChunks++
		s.mu.Unlock()

		nodePath := filepath.Join(s.basePath, fmt.Sprintf("n%d", nodeNum))
		chunkPath := filepath.Join(nodePath, hash[:16])

		if err := os.WriteFile(chunkPath, chunk, 0644); err != nil {
			errorChan <- fmt.Errorf("failed to write chunk: %w", err)
			wg.Done()
			return
		}

		wg.Done()
	}
}

// GetNodeStatistics implements domain.StorageService.
func (s *storageService) GetNodeStatistics() []models.NodeStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make([]models.NodeStatus, len(s.nodes))
	copy(stats, s.nodes)
	return stats

}

// HashData implements domain.StorageService.
func (s *storageService) HashData(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Initialize implements domain.StorageService.
func (s *storageService) Initialize() error {
	for i := 0; i < s.numNodes; i++ {
		nodePath := filepath.Join(s.basePath, fmt.Sprintf("n%d", i))
		if err := os.MkdirAll(nodePath, 0755); err != nil {
			return fmt.Errorf("failed to create node directory: %w", err)
		}

		// Calculate existing usage
		if err := s.UpdateNodeUsage(uint8(i)); err != nil {
			return fmt.Errorf("failed to calculate node usage: %w", err)
		}
	}
	return nil
}

// ReconstructFile implements domain.StorageService.
func (s *storageService) ReconstructFile(metadata *models.FileMetadata, outputPath string) error {
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Sort chunks by index to ensure correct order
	sortedChunks := make([]models.ChunkInfo, len(metadata.Chunks))
	copy(sortedChunks, metadata.Chunks)
	sort.Slice(sortedChunks, func(i, j int) bool {
		return sortedChunks[i].I < sortedChunks[j].I
	})

	for _, chunk := range sortedChunks {
		chunkPath := filepath.Join(s.basePath, fmt.Sprintf("n%d", chunk.N), chunk.H)
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			return fmt.Errorf("failed to read chunk: %w", err)
		}

		if _, err := outFile.Write(data); err != nil {
			return fmt.Errorf("failed to write to output file: %w", err)
		}
	}

	return nil
}

// SaveMetadataEfficient implements domain.StorageService.
func (s *storageService) SaveMetadataEfficient(metadata *models.FileMetadata) error {
	buf := make([]byte, 0, 1024)

	// Write file size
	sizeBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(sizeBytes, uint64(metadata.Size))
	buf = append(buf, sizeBytes...)

	// Write name length and name
	nameBytes := []byte(metadata.Name)
	nameLenBytes := make([]byte, 2)
	binary.LittleEndian.PutUint16(nameLenBytes, uint16(len(nameBytes)))
	buf = append(buf, nameLenBytes...)
	buf = append(buf, nameBytes...)

	// Write type
	buf = append(buf, []byte(metadata.Type)...)

	// Write number of chunks
	numChunksBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(numChunksBytes, uint32(len(metadata.Chunks)))
	buf = append(buf, numChunksBytes...)

	// Write chunks information
	for _, chunk := range metadata.Chunks {
		indexBytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(indexBytes, chunk.I)
		buf = append(buf, indexBytes...)

		buf = append(buf, chunk.N)
		buf = append(buf, []byte(chunk.H)...)

		// Write chunk size
		chunkSizeBytes := make([]byte, 8)
		binary.LittleEndian.PutUint64(chunkSizeBytes, uint64(chunk.S))
		buf = append(buf, chunkSizeBytes...)
	}

	metadataPath := filepath.Join(s.basePath, "meta.bin")
	if err := os.WriteFile(metadataPath, buf, 0644); err != nil {
		return fmt.Errorf("failed to write binary metadata: %w", err)
	}

	// Save JSON version for debugging
	jsonPath := filepath.Join(s.basePath, "meta.json")
	jsonData, _ := json.MarshalIndent(metadata, "", "  ")
	os.WriteFile(jsonPath, jsonData, 0644)

	return nil
}

// SelectOptimalNode implements domain.StorageService.
func (s *storageService) SelectOptimalNode(chunkSize int64) uint8 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	availableNodes := make([]models.NodeStatus, 0, len(s.nodes))
	for _, node := range s.nodes {
		if node.Available {
			availableNodes = append(availableNodes, node)
		}
	}

	sort.Slice(availableNodes, func(i, j int) bool {
		if abs(availableNodes[i].UsedSpace-availableNodes[j].UsedSpace) > 1024*1024*10 {
			return availableNodes[i].UsedSpace < availableNodes[j].UsedSpace
		}
		return availableNodes[i].NumChunks < availableNodes[j].NumChunks
	})

	return availableNodes[0].NodeID
}

// StoreFile implements domain.StorageService.
func (s *storageService) StoreFile(filePath string) (*models.FileMetadata, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	metadata := &models.FileMetadata{
		Name: filepath.Base(filePath),
		Type: filepath.Ext(filePath),
		Size: fileInfo.Size(),
	}

	expectedChunks := int(fileInfo.Size())/s.chunkSize + 1
	metadata.Chunks = make([]models.ChunkInfo, 0, expectedChunks)

	var wg sync.WaitGroup
	chunksChan := make(chan []byte, 5)
	errorChan := make(chan error, 1)

	for i := 0; i < 3; i++ {
		go s.ChunkWorker(&wg, chunksChan, errorChan, metadata)
	}

	buffer := make([]byte, s.chunkSize)
	var chunkIndex uint32 = 0
	for {
		n, err := file.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to read chunk: %w", err)
		}

		chunk := make([]byte, n)
		copy(chunk, buffer[:n])

		wg.Add(1)
		chunksChan <- chunk
		chunkIndex++
	}

	close(chunksChan)
	wg.Wait()

	select {
	case err := <-errorChan:
		return nil, err
	default:
	}

	if err := s.SaveMetadataEfficient(metadata); err != nil {
		return nil, fmt.Errorf("failed to save metadata: %w", err)
	}

	return metadata, nil
}

// UpdateNodeUsage implements domain.StorageService.
func (s *storageService) UpdateNodeUsage(nodeID uint8) error {
	nodePath := filepath.Join(s.basePath, fmt.Sprintf("n%d", nodeID))
	var totalSize int64
	var numChunks int

	err := filepath.Walk(nodePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
			numChunks++
		}
		return nil
	})

	if err != nil {
		return err
	}

	s.mu.Lock()
	s.nodes[nodeID].UsedSpace = totalSize
	s.nodes[nodeID].NumChunks = numChunks
	s.mu.Unlock()

	return nil
}

func NewStorageService(basePath string, chunkSize int, numNodes int) domain.StorageService {
	nodes := make([]models.NodeStatus, numNodes)
	for i := 0; i < numNodes; i++ {
		nodes[i] = models.NodeStatus{
			NodeID:    uint8(i),
			Available: true,
		}
	}
	return &storageService{
		basePath:  basePath,
		chunkSize: chunkSize,
		numNodes:  numNodes,
		nodes:     nodes,
	}

}
func abs(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}
