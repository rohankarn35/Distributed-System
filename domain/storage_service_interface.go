package domain

import (
	"distributed-cas/models"
	"sync"
)

type StorageService interface {
	Initialize() error
	UpdateNodeUsage(nodeID uint8) error
	SelectOptimalNode(chunkSize int64) uint8
	StoreFile(filePath string) (*models.FileMetadata, error)
	ChunkWorker(wg *sync.WaitGroup, chunks <-chan []byte, errorChan chan<- error, metadata *models.FileMetadata)
	SaveMetadataEfficient(metadata *models.FileMetadata) error
	ReconstructFile(metadata *models.FileMetadata, outputPath string) error
	HashData(data []byte) string
	GetNodeStatistics() []models.NodeStatus
}
