package protocol

import (
	"encoding/json"
	"time"
)

// ControlMessage represents a Celery control message
type ControlMessage struct {
	Method      string                 `json:"method"`
	Arguments   map[string]interface{} `json:"arguments"`
	Destination []string               `json:"destination,omitempty"`
	Reply       bool                   `json:"reply,omitempty"`
	Ticket      string                 `json:"ticket,omitempty"`
}

// PingMessage represents a ping control message
type PingMessage struct {
	ControlMessage
}

// NewPingMessage creates a new ping control message
func NewPingMessage() *PingMessage {
	return &PingMessage{
		ControlMessage: ControlMessage{
			Method:    "ping",
			Arguments: make(map[string]interface{}),
			Reply:     true,
		},
	}
}

// ToJSON serializes the message to JSON
func (pm *PingMessage) ToJSON() ([]byte, error) {
	return json.Marshal(pm.ControlMessage)
}

// PingResponse represents a response to a ping message
type PingResponse struct {
	Method    string                 `json:"method"`
	Arguments map[string]interface{} `json:"arguments"`
	Hostname  string                 `json:"hostname"`
	Timestamp float64                `json:"timestamp"`
	Ticket    string                 `json:"ticket,omitempty"`
}

// WorkerInfo represents information about a Celery worker
type WorkerInfo struct {
	Hostname  string    `json:"hostname"`
	Timestamp time.Time `json:"timestamp"`
	Active    bool      `json:"active"`
	Processed int       `json:"processed"`
	LoadAvg   []float64 `json:"loadavg,omitempty"`
}

// ParsePingResponse parses a JSON response into a PingResponse
func ParsePingResponse(data []byte) (*PingResponse, error) {
	var response PingResponse
	err := json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// CeleryMessage represents the basic structure of Celery messages
type CeleryMessage struct {
	Body            string                 `json:"body"`
	ContentType     string                 `json:"content-type"`
	ContentEncoding string                 `json:"content-encoding"`
	Headers         map[string]interface{} `json:"headers"`
	Properties      MessageProperties      `json:"properties"`
}

// MessageProperties represents message properties
type MessageProperties struct {
	CorrelationID string `json:"correlation_id"`
	ReplyTo       string `json:"reply_to,omitempty"`
	DeliveryMode  int    `json:"delivery_mode"`
	DeliveryInfo  map[string]interface{} `json:"delivery_info"`
	Priority      int    `json:"priority"`
	BodyEncoding  string `json:"body_encoding"`
}

// BroadcastMessage represents a broadcast control message
type BroadcastMessage struct {
	Pattern   string      `json:"pattern,omitempty"`
	Matcher   string      `json:"matcher,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp float64     `json:"timestamp"`
}

// NewBroadcastMessage creates a new broadcast message
func NewBroadcastMessage(data interface{}) *BroadcastMessage {
	return &BroadcastMessage{
		Data:      data,
		Timestamp: float64(time.Now().Unix()),
	}
}