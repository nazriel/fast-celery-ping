package protocol

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MessageFormat represents the format of the ping message
type MessageFormat int

const (
	// MessageFormatRaw returns the control message as raw JSON (used by AMQP)
	MessageFormatRaw MessageFormat = iota
	// MessageFormatEnveloped returns the control message base64-encoded and wrapped in envelope (used by Redis)
	MessageFormatEnveloped
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

// CreatePingMessage creates a Celery ping message in the specified format
func (h *Handler) CreatePingMessage(replyTo string, destinations []string, format MessageFormat) ([]byte, error) {
	ticket := uuid.New().String()

	// Determine destination - nil for broadcast, or specific destinations
	var destination interface{}
	if len(destinations) > 0 {
		destination = destinations
	} else {
		destination = nil
	}

	// Create the control message that Celery workers expect
	controlMessage := map[string]interface{}{
		"method":      "ping",
		"arguments":   map[string]interface{}{},
		"destination": destination,
		"pattern":     nil,
		"matcher":     nil,
		"ticket":      ticket,
		"reply_to": map[string]interface{}{
			"exchange":    "reply.celery.pidbox",
			"routing_key": replyTo,
		},
	}

	// Apply format-specific processing
	switch format {
	case MessageFormatRaw:
		// Return the control message directly as JSON (used by AMQP)
		return json.Marshal(controlMessage)
	case MessageFormatEnveloped:
		// Base64 encode the control message and wrap in envelope (used by Redis)
		bodyBytes, err := json.Marshal(controlMessage)
		if err != nil {
			return nil, err
		}

		// Base64 encode the body like Python Celery does
		base64Body := base64.StdEncoding.EncodeToString(bodyBytes)

		// Set expiry to 10 seconds to ensure workers have time to respond
		now := time.Now()
		expires := now.Add(10 * time.Second).Unix()

		// Create the complete message envelope matching Python Celery exactly
		envelope := map[string]interface{}{
			"body":             base64Body,
			"content-encoding": "utf-8",
			"content-type":     "application/json",
			"headers": map[string]interface{}{
				"clock":   1,
				"expires": expires,
			},
			"properties": map[string]interface{}{
				"delivery_mode": 2,
				"delivery_info": map[string]interface{}{
					"exchange":    "celery.pidbox",
					"routing_key": "",
				},
				"priority":      0,
				"body_encoding": "base64",
				"delivery_tag":  uuid.New().String(),
			},
		}

		return json.Marshal(envelope)
	default:
		return nil, fmt.Errorf("unsupported message format: %v", format)
	}
}

// ParseWorkerResponse parses a worker response and extracts relevant information
func (h *Handler) ParseWorkerResponse(data []byte) (map[string]interface{}, error) {
	var envelope map[string]interface{}

	// Parse the response envelope
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, fmt.Errorf("failed to parse response envelope: %w", err)
	}

	// Check if there's a base64-encoded body
	if bodyStr, exists := envelope["body"]; exists {
		if bodyString, ok := bodyStr.(string); ok {
			// Decode base64 body
			bodyBytes, err := base64.StdEncoding.DecodeString(bodyString)
			if err != nil {
				return nil, fmt.Errorf("failed to decode base64 body: %w", err)
			}

			// Parse the decoded body as JSON
			var decodedBody map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &decodedBody); err != nil {
				return nil, fmt.Errorf("failed to parse decoded body: %w", err)
			}

			// Return the decoded body as the main response
			return decodedBody, nil
		}
	}

	// Fallback: return the envelope as-is
	return envelope, nil
}

// ExtractWorkerName extracts worker name from various response formats
func (h *Handler) ExtractWorkerName(response map[string]interface{}) string {
	// For worker responses, look for keys that contain @ (worker names)
	for workerName, value := range response {
		if strings.Contains(workerName, "@") {
			// Verify this is a worker response by checking for "ok" field
			if workerData, ok := value.(map[string]interface{}); ok {
				if _, exists := workerData["ok"]; exists {
					return workerName
				}
			}
		}
	}

	// Try different fields that might contain the worker name
	fields := []string{"hostname", "worker", "nodename", "node", "name"}

	// Check in the data field first
	if data, exists := response["data"]; exists {
		if dataMap, ok := data.(map[string]interface{}); ok {
			for _, field := range fields {
				if value, exists := dataMap[field]; exists {
					if strValue, ok := value.(string); ok && strValue != "" {
						return strValue
					}
				}
			}
		}
	}

	// Check at top level
	for _, field := range fields {
		if value, exists := response[field]; exists {
			if strValue, ok := value.(string); ok && strValue != "" {
				return strValue
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
	// For worker responses, check if any key contains an "ok" field with "pong"
	for workerName, value := range response {
		if strings.Contains(workerName, "@") { // worker names typically contain @
			if workerData, ok := value.(map[string]interface{}); ok {
				if status, exists := workerData["ok"]; exists {
					if statusStr, ok := status.(string); ok && statusStr == "pong" {
						return true
					}
				}
			}
		}
	}

	// Check for worker information in various locations
	if hostname := h.ExtractWorkerName(response); hostname != "" {
		return true
	}

	return false
}

// CreateReplyQueue generates a unique reply queue name
func (h *Handler) CreateReplyQueue() string {
	// Use simple UUID format like Python Celery does
	return uuid.New().String()
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
