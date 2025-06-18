package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestNewPingMessage(t *testing.T) {
	msg := NewPingMessage()

	if msg == nil {
		t.Fatal("Expected non-nil ping message")
	}

	if msg.Method != "ping" {
		t.Errorf("Expected method 'ping', got %s", msg.Method)
	}

	if msg.Arguments == nil {
		t.Error("Expected non-nil arguments map")
	}

	if !msg.Reply {
		t.Error("Expected reply to be true")
	}
}

func TestPingMessage_ToJSON(t *testing.T) {
	msg := NewPingMessage()
	msg.Destination = []string{"worker1@host", "worker2@host"}
	msg.Ticket = "test-ticket"

	data, err := msg.ToJSON()
	if err != nil {
		t.Fatalf("Failed to convert to JSON: %v", err)
	}

	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if parsed["method"] != "ping" {
		t.Errorf("Expected method 'ping', got %v", parsed["method"])
	}

	if parsed["reply"] != true {
		t.Errorf("Expected reply true, got %v", parsed["reply"])
	}

	if parsed["ticket"] != "test-ticket" {
		t.Errorf("Expected ticket 'test-ticket', got %v", parsed["ticket"])
	}

	// Check destination array
	if dest, ok := parsed["destination"].([]interface{}); ok {
		if len(dest) != 2 {
			t.Errorf("Expected 2 destinations, got %d", len(dest))
		}
		if dest[0] != "worker1@host" || dest[1] != "worker2@host" {
			t.Errorf("Expected specific destinations, got %v", dest)
		}
	} else {
		t.Error("Expected destination to be an array")
	}
}

func TestParsePingResponse(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
		expected *PingResponse
	}{
		{
			name:     "valid response",
			jsonData: `{"method":"pong","arguments":{},"hostname":"worker@host","timestamp":1234567890,"ticket":"test-ticket"}`,
			wantErr:  false,
			expected: &PingResponse{
				Method:    "pong",
				Arguments: map[string]interface{}{},
				Hostname:  "worker@host",
				Timestamp: 1234567890,
				Ticket:    "test-ticket",
			},
		},
		{
			name:     "minimal response",
			jsonData: `{"method":"pong"}`,
			wantErr:  false,
			expected: &PingResponse{
				Method:    "pong",
				Arguments: nil,
				Hostname:  "",
				Timestamp: 0,
				Ticket:    "",
			},
		},
		{
			name:     "invalid JSON",
			jsonData: `{"method":"pong"`,
			wantErr:  true,
			expected: nil,
		},
		{
			name:     "empty JSON",
			jsonData: `{}`,
			wantErr:  false,
			expected: &PingResponse{
				Method:    "",
				Arguments: nil,
				Hostname:  "",
				Timestamp: 0,
				Ticket:    "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ParsePingResponse([]byte(tt.jsonData))

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if result == nil {
				t.Fatal("Expected non-nil result")
			}

			if result.Method != tt.expected.Method {
				t.Errorf("Expected method %s, got %s", tt.expected.Method, result.Method)
			}

			if result.Hostname != tt.expected.Hostname {
				t.Errorf("Expected hostname %s, got %s", tt.expected.Hostname, result.Hostname)
			}

			if result.Timestamp != tt.expected.Timestamp {
				t.Errorf("Expected timestamp %f, got %f", tt.expected.Timestamp, result.Timestamp)
			}

			if result.Ticket != tt.expected.Ticket {
				t.Errorf("Expected ticket %s, got %s", tt.expected.Ticket, result.Ticket)
			}
		})
	}
}

func TestNewBroadcastMessage(t *testing.T) {
	testData := map[string]interface{}{
		"test": "data",
		"num":  42,
	}

	// Record time before creating message
	beforeTime := time.Now().Unix()

	msg := NewBroadcastMessage(testData)

	// Record time after creating message
	afterTime := time.Now().Unix()

	if msg == nil {
		t.Fatal("Expected non-nil broadcast message")
	}

	// Compare data by converting to JSON since maps can't be directly compared
	if msg.Data == nil {
		t.Error("Expected data to be set")
	}

	dataMap, ok := msg.Data.(map[string]interface{})
	if !ok {
		t.Error("Expected data to be a map")
	} else {
		if dataMap["test"] != "data" || dataMap["num"] != 42 {
			t.Error("Expected data to match input")
		}
	}

	// Check timestamp is reasonable (within the time window)
	timestampInt := int64(msg.Timestamp)
	if timestampInt < beforeTime || timestampInt > afterTime {
		t.Errorf("Expected timestamp between %d and %d, got %d", beforeTime, afterTime, timestampInt)
	}

	// Pattern and Matcher should be empty by default
	if msg.Pattern != "" {
		t.Errorf("Expected empty pattern, got %s", msg.Pattern)
	}

	if msg.Matcher != "" {
		t.Errorf("Expected empty matcher, got %s", msg.Matcher)
	}
}

func TestBroadcastMessage_JSON_Serialization(t *testing.T) {
	testData := map[string]interface{}{
		"command": "status",
		"args":    []string{"--verbose"},
	}

	msg := NewBroadcastMessage(testData)
	msg.Pattern = "worker.*"
	msg.Matcher = "glob"

	// Test that the message can be serialized to JSON
	jsonData, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal broadcast message: %v", err)
	}

	// Test that it can be deserialized back
	var parsed BroadcastMessage
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal broadcast message: %v", err)
	}

	if parsed.Pattern != msg.Pattern {
		t.Errorf("Expected pattern %s, got %s", msg.Pattern, parsed.Pattern)
	}

	if parsed.Matcher != msg.Matcher {
		t.Errorf("Expected matcher %s, got %s", msg.Matcher, parsed.Matcher)
	}

	if parsed.Timestamp != msg.Timestamp {
		t.Errorf("Expected timestamp %f, got %f", msg.Timestamp, parsed.Timestamp)
	}

	// Check that data is preserved
	parsedData, ok := parsed.Data.(map[string]interface{})
	if !ok {
		t.Fatal("Expected data to be a map")
	}

	if parsedData["command"] != "status" {
		t.Errorf("Expected command 'status', got %v", parsedData["command"])
	}
}

func TestControlMessage_JSON_Serialization(t *testing.T) {
	msg := ControlMessage{
		Method:      "inspect",
		Arguments:   map[string]interface{}{"type": "stats"},
		Destination: []string{"worker1@host"},
		Reply:       true,
		Ticket:      "inspect-ticket",
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal control message: %v", err)
	}

	var parsed ControlMessage
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal control message: %v", err)
	}

	if parsed.Method != msg.Method {
		t.Errorf("Expected method %s, got %s", msg.Method, parsed.Method)
	}

	if parsed.Reply != msg.Reply {
		t.Errorf("Expected reply %v, got %v", msg.Reply, parsed.Reply)
	}

	if parsed.Ticket != msg.Ticket {
		t.Errorf("Expected ticket %s, got %s", msg.Ticket, parsed.Ticket)
	}

	if len(parsed.Destination) != 1 || parsed.Destination[0] != "worker1@host" {
		t.Errorf("Expected destination [worker1@host], got %v", parsed.Destination)
	}
}

func TestCeleryMessage_JSON_Serialization(t *testing.T) {
	msg := CeleryMessage{
		Body:            "base64encodedcontent",
		ContentType:     "application/json",
		ContentEncoding: "utf-8",
		Headers: map[string]interface{}{
			"task":    "myapp.tasks.add",
			"expires": 1234567890,
		},
		Properties: MessageProperties{
			CorrelationID: "correlation-123",
			ReplyTo:       "reply-queue",
			DeliveryMode:  2,
			DeliveryInfo: map[string]interface{}{
				"exchange":    "celery",
				"routing_key": "celery",
			},
			Priority:     0,
			BodyEncoding: "base64",
		},
	}

	jsonData, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("Failed to marshal celery message: %v", err)
	}

	var parsed CeleryMessage
	err = json.Unmarshal(jsonData, &parsed)
	if err != nil {
		t.Fatalf("Failed to unmarshal celery message: %v", err)
	}

	if parsed.Body != msg.Body {
		t.Errorf("Expected body %s, got %s", msg.Body, parsed.Body)
	}

	if parsed.ContentType != msg.ContentType {
		t.Errorf("Expected content type %s, got %s", msg.ContentType, parsed.ContentType)
	}

	if parsed.Properties.CorrelationID != msg.Properties.CorrelationID {
		t.Errorf("Expected correlation ID %s, got %s", msg.Properties.CorrelationID, parsed.Properties.CorrelationID)
	}

	if parsed.Properties.DeliveryMode != msg.Properties.DeliveryMode {
		t.Errorf("Expected delivery mode %d, got %d", msg.Properties.DeliveryMode, parsed.Properties.DeliveryMode)
	}
}
