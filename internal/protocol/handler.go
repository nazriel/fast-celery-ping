package protocol

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Handler manages Celery protocol operations
type Handler struct {
	nodeID string
}

// NewHandler creates a new protocol handler
func NewHandler() *Handler {
	return &Handler{
		nodeID: fmt.Sprintf("fast-celery-ping@%s", generateHostname()),
	}
}

// CreatePingMessage creates a properly formatted Celery ping message
func (h *Handler) CreatePingMessage(replyTo string) ([]byte, error) {
	ticket := uuid.New().String()
	
	message := map[string]interface{}{
		"method":      "ping",
		"arguments":   map[string]interface{}{},
		"destination": nil,
		"reply":       true,
		"ticket":      ticket,
	}

	// Wrap in broadcast format
	broadcast := map[string]interface{}{
		"data":      message,
		"timestamp": float64(time.Now().Unix()),
	}

	return json.Marshal(broadcast)
}

// ParseWorkerResponse parses a worker response and extracts relevant information
func (h *Handler) ParseWorkerResponse(data []byte) (map[string]interface{}, error) {
	var response map[string]interface{}
	
	// Try parsing as direct JSON first
	if err := json.Unmarshal(data, &response); err != nil {
		// If that fails, try parsing as string
		var strResponse string
		if err := json.Unmarshal(data, &strResponse); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}
		
		// Try parsing the string content as JSON
		if err := json.Unmarshal([]byte(strResponse), &response); err != nil {
			// If all parsing fails, return the string as a simple response
			return map[string]interface{}{
				"raw": strResponse,
			}, nil
		}
	}
	
	return response, nil
}

// ExtractWorkerName extracts worker name from various response formats
func (h *Handler) ExtractWorkerName(response map[string]interface{}) string {
	// Try different fields that might contain the worker name
	fields := []string{"hostname", "worker", "nodename", "node", "name"}
	
	for _, field := range fields {
		if value, exists := response[field]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				return strValue
			}
		}
	}
	
	// Check if there's a nested worker info
	if worker, exists := response["worker"]; exists {
		if workerMap, ok := worker.(map[string]interface{}); ok {
			for _, field := range fields {
				if value, exists := workerMap[field]; exists {
					if strValue, ok := value.(string); ok && strValue != "" {
						return strValue
					}
				}
			}
		}
	}
	
	// As a last resort, look for any string field that looks like a hostname
	for key, value := range response {
		if strValue, ok := value.(string); ok {
			if strings.Contains(strValue, "@") || strings.Contains(key, "host") {
				return strValue
			}
		}
	}
	
	return ""
}

// ValidateResponse checks if a response is a valid ping response
func (h *Handler) ValidateResponse(response map[string]interface{}) bool {
	// Check for basic response structure
	if method, exists := response["method"]; exists {
		if methodStr, ok := method.(string); ok && methodStr == "pong" {
			return true
		}
	}
	
	// Check for worker information
	if hostname := h.ExtractWorkerName(response); hostname != "" {
		return true
	}
	
	// Check for common Celery response patterns
	if _, exists := response["hostname"]; exists {
		return true
	}
	
	if _, exists := response["worker"]; exists {
		return true
	}
	
	return false
}

// CreateReplyQueue generates a unique reply queue name
func (h *Handler) CreateReplyQueue() string {
	return fmt.Sprintf("_kombu.binding.reply.celery.pidbox.%s", uuid.New().String())
}

// GetBroadcastQueue returns the broadcast queue name for ping messages
func (h *Handler) GetBroadcastQueue() string {
	return "celeryctl-broadcast-pidbox"
}

// generateHostname creates a hostname for this instance
func generateHostname() string {
	// In a real implementation, you might want to get the actual hostname
	// For now, use a UUID-based identifier
	return fmt.Sprintf("host-%s", uuid.New().String()[:8])
}

// FormatResponse formats the response in the expected Celery format
func (h *Handler) FormatResponse(workerName, status string, timestamp time.Time) map[string]interface{} {
	return map[string]interface{}{
		workerName: map[string]interface{}{
			"ok": status,
		},
	}
}