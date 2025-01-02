package models

// NodeStatus tracks node storage capacity and usage
type NodeStatus struct {
	NodeID    uint8
	UsedSpace int64
	NumChunks int
	Available bool
}

// ChunkInfo with added size field for better allocation decisions
type ChunkInfo struct {
	H string `json:"h,omitempty"` // hash
	I uint32 `json:"i"`           // index
	N uint8  `json:"n"`           // node number
	S int64  `json:"s"`           // chunk size
}

type FileMetadata struct {
	Name   string      `json:"n,omitempty"`
	Type   string      `json:"t,omitempty"`
	Size   int64       `json:"s,omitempty"`
	Chunks []ChunkInfo `json:"c,omitempty"`
}
