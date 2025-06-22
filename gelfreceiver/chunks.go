package gelfreceiver

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ChunkInfo represents information about a chunked message
type ChunkInfo struct {
	messageID      [8]byte
	totalChunks    int
	receivedChunks map[int][]byte
	timestamp      time.Time
}

// ChunkAssembler handles reassembly of chunked GELF messages over UDP
type ChunkAssembler struct {
	chunks        map[[8]byte]*ChunkInfo
	mutex         sync.RWMutex
	timeout       time.Duration
	logger        *zap.Logger
	cleanupTicker *time.Ticker
	stopCleanup   chan struct{}
}

// NewChunkAssembler creates a new chunk assembler
func NewChunkAssembler(timeout time.Duration, logger *zap.Logger) *ChunkAssembler {
	assembler := &ChunkAssembler{
		chunks:      make(map[[8]byte]*ChunkInfo),
		timeout:     timeout,
		logger:      logger,
		stopCleanup: make(chan struct{}),
	}

	// Start cleanup goroutine
	assembler.cleanupTicker = time.NewTicker(timeout / 2)
	go assembler.cleanupExpiredChunks()

	return assembler
}

// AddChunk adds a chunk to the assembler and returns the complete message if all chunks are received
func (ca *ChunkAssembler) AddChunk(data []byte) ([]byte, error) {
	if len(data) < 12 {
		return nil, fmt.Errorf("chunk too small: %d bytes", len(data))
	}

	// Verify magic bytes
	if data[0] != 0x1e || data[1] != 0x0f {
		return nil, fmt.Errorf("invalid chunk magic bytes: %02x %02x", data[0], data[1])
	}

	// Extract chunk header
	var messageID [8]byte
	copy(messageID[:], data[2:10])
	sequenceNum := int(data[10])
	sequenceCount := int(data[11])

	// Validate sequence numbers
	if sequenceNum >= sequenceCount {
		return nil, fmt.Errorf("invalid sequence number: %d >= %d", sequenceNum, sequenceCount)
	}

	if sequenceCount > 128 {
		return nil, fmt.Errorf("too many chunks: %d (max 128)", sequenceCount)
	}

	// Extract payload
	payload := data[12:]

	ca.mutex.Lock()
	defer ca.mutex.Unlock()

	// Get or create chunk info
	chunkInfo, exists := ca.chunks[messageID]
	if !exists {
		chunkInfo = &ChunkInfo{
			messageID:      messageID,
			totalChunks:    sequenceCount,
			receivedChunks: make(map[int][]byte),
			timestamp:      time.Now(),
		}
		ca.chunks[messageID] = chunkInfo
	}

	// Validate total chunks consistency
	if chunkInfo.totalChunks != sequenceCount {
		return nil, fmt.Errorf("inconsistent chunk count for message %x: expected %d, got %d",
			messageID, chunkInfo.totalChunks, sequenceCount)
	}

	// Add the chunk
	chunkInfo.receivedChunks[sequenceNum] = payload

	ca.logger.Debug("Received chunk",
		zap.String("message_id", fmt.Sprintf("%x", messageID)),
		zap.Int("sequence", sequenceNum),
		zap.Int("total_chunks", sequenceCount),
		zap.Int("received_chunks", len(chunkInfo.receivedChunks)))

	// Check if we have all chunks
	if len(chunkInfo.receivedChunks) == chunkInfo.totalChunks {
		// Assemble the complete message
		complete := ca.assembleMessage(chunkInfo)

		// Remove from chunks map
		delete(ca.chunks, messageID)

		ca.logger.Debug("Message assembly complete",
			zap.String("message_id", fmt.Sprintf("%x", messageID)),
			zap.Int("total_size", len(complete)))

		return complete, nil
	}

	// Message not complete yet
	return nil, nil
}

// assembleMessage assembles chunks into a complete message
func (ca *ChunkAssembler) assembleMessage(chunkInfo *ChunkInfo) []byte {
	var buffer bytes.Buffer

	// Assemble chunks in order
	for i := 0; i < chunkInfo.totalChunks; i++ {
		chunk, exists := chunkInfo.receivedChunks[i]
		if !exists {
			ca.logger.Error("Missing chunk during assembly",
				zap.String("message_id", fmt.Sprintf("%x", chunkInfo.messageID)),
				zap.Int("missing_chunk", i))
			continue
		}
		buffer.Write(chunk)
	}

	return buffer.Bytes()
}

// cleanupExpiredChunks removes expired chunks to prevent memory leaks
func (ca *ChunkAssembler) cleanupExpiredChunks() {
	for {
		select {
		case <-ca.cleanupTicker.C:
			ca.mutex.Lock()
			now := time.Now()
			expiredCount := 0

			for messageID, chunkInfo := range ca.chunks {
				if now.Sub(chunkInfo.timestamp) > ca.timeout {
					delete(ca.chunks, messageID)
					expiredCount++
				}
			}

			if expiredCount > 0 {
				ca.logger.Debug("Cleaned up expired chunks",
					zap.Int("expired_messages", expiredCount),
					zap.Int("remaining_messages", len(ca.chunks)))
			}

			ca.mutex.Unlock()

		case <-ca.stopCleanup:
			return
		}
	}
}

// Stop stops the chunk assembler and cleans up resources
func (ca *ChunkAssembler) Stop() {
	if ca.cleanupTicker != nil {
		ca.cleanupTicker.Stop()
	}

	close(ca.stopCleanup)

	ca.mutex.Lock()
	defer ca.mutex.Unlock()

	// Clear all chunks
	ca.chunks = make(map[[8]byte]*ChunkInfo)
}
