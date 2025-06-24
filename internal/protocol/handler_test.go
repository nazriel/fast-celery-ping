package protocol

import (
	"encoding/json"
	"testing"
	"time"
)

func TestHandler_NewHandler(t *testing.T) {
	handler := NewHandler()

	if handler == nil {
		t.Fatal("Expected handler to be created, got nil")
	}

	if handler.nodeID == "" {
		t.Fatal("Expected nodeID to be set")
	}
}

func TestHandler_CreateReplyQueue(t *testing.T) {
	handler := NewHandler()

	queue1 := handler.CreateReplyQueue()
	queue2 := handler.CreateReplyQueue()

	if queue1 == queue2 {
		t.Error("Expected different queue names for each call")
	}

	if queue1 == "" || queue2 == "" {
		t.Error("Expected non-empty queue names")
	}
}

func TestHandler_GetBroadcastQueue(t *testing.T) {
	handler := NewHandler()

	queue := handler.GetBroadcastQueue()
	expected := "celeryctl-broadcast-pidbox"

	if queue != expected {
		t.Errorf("Expected broadcast queue %s, got %s", expected, queue)
	}
}

func TestHandler_CreatePingMessage(t *testing.T) {
	handler := NewHandler()
	replyTo := "reply-queue-test"

	tests := []struct {
		name         string
		destinations []string
		format       MessageFormat
		checkFields  []string
	}{
		{
			name:         "broadcast message enveloped",
			destinations: nil,
			format:       MessageFormatEnveloped,
			checkFields:  []string{"body", "properties", "headers"},
		},
		{
			name:         "targeted message enveloped",
			destinations: []string{"worker1@host", "worker2@host"},
			format:       MessageFormatEnveloped,
			checkFields:  []string{"body", "properties", "headers"},
		},
		{
			name:         "broadcast message raw",
			destinations: nil,
			format:       MessageFormatRaw,
			checkFields:  []string{"method", "arguments", "ticket", "reply_to"},
		},
		{
			name:         "targeted message raw",
			destinations: []string{"worker1@host", "worker2@host"},
			format:       MessageFormatRaw,
			checkFields:  []string{"method", "arguments", "ticket", "reply_to"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageData, err := handler.CreatePingMessage(replyTo, tt.destinations, tt.format)
			if err != nil {
				t.Fatalf("Failed to create ping message: %v", err)
			}

			var message map[string]interface{}
			err = json.Unmarshal(messageData, &message)
			if err != nil {
				t.Fatalf("Failed to unmarshal ping message: %v", err)
			}

			// Check that the message has the expected structure
			for _, field := range tt.checkFields {
				if _, exists := message[field]; !exists {
					t.Errorf("Expected '%s' field in ping message", field)
				}
			}
		})
	}
}

func TestHandler_ExtractWorkerName(t *testing.T) {
	handler := NewHandler()

	tests := []struct {
		name     string
		response map[string]interface{}
		expected string
	}{
		{
			name: "celery worker response format",
			response: map[string]interface{}{
				"celery@nero": map[string]interface{}{
					"ok": "pong",
				},
			},
			expected: "celery@nero",
		},
		{
			name: "hostname field fallback",
			response: map[string]interface{}{
				"hostname": "worker1@host",
			},
			expected: "worker1@host",
		},
		{
			name: "worker field fallback",
			response: map[string]interface{}{
				"worker": "worker2@host",
			},
			expected: "worker2@host",
		},
		{
			name: "no worker name",
			response: map[string]interface{}{
				"other": "data",
			},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ExtractWorkerName(tt.response)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHandler_ValidateResponse(t *testing.T) {
	handler := NewHandler()

	tests := []struct {
		name     string
		response map[string]interface{}
		expected bool
	}{
		{
			name: "valid celery pong response",
			response: map[string]interface{}{
				"celery@nero": map[string]interface{}{
					"ok": "pong",
				},
			},
			expected: true,
		},
		{
			name: "response with hostname fallback",
			response: map[string]interface{}{
				"hostname": "worker@host",
			},
			expected: true,
		},
		{
			name: "response with worker fallback",
			response: map[string]interface{}{
				"worker": "worker@host",
			},
			expected: true,
		},
		{
			name: "invalid response",
			response: map[string]interface{}{
				"other": "data",
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ValidateResponse(tt.response)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestHandler_ParseWorkerResponse(t *testing.T) {
	handler := NewHandler()

	tests := []struct {
		name        string
		data        []byte
		wantErr     bool
		expectedLen int
		checkField  func(map[string]interface{}) bool
	}{
		{
			name: "base64 encoded celery response",
			data: []byte(`{
				"body": "eyJjZWxlcnlAaG9zdCI6IHsib2siOiAicG9uZyJ9fQ==",
				"properties": {"delivery_mode": 2},
				"headers": {"expires": 1234567890}
			}`),
			wantErr:     false,
			expectedLen: 1,
			checkField: func(response map[string]interface{}) bool {
				if workerData, exists := response["celery@host"]; exists {
					if workerMap, ok := workerData.(map[string]interface{}); ok {
						return workerMap["ok"] == "pong"
					}
				}
				return false
			},
		},
		{
			name: "direct JSON response without base64",
			data: []byte(`{
				"celery@worker": {"ok": "pong"},
				"hostname": "worker@host"
			}`),
			wantErr:     false,
			expectedLen: 2,
			checkField: func(response map[string]interface{}) bool {
				return response["hostname"] == "worker@host"
			},
		},
		{
			name:    "invalid JSON",
			data:    []byte(`{"invalid": "json"`),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte(``),
			wantErr: true,
		},
		{
			name: "base64 with invalid inner JSON",
			data: []byte(`{
				"body": "aW52YWxpZCBqc29u",
				"properties": {}
			}`),
			wantErr: true,
		},
		{
			name: "base64 with valid inner JSON",
			data: []byte(`{
				"body": "eyJ3b3JrZXIxQGhvc3QiOiB7Im9rIjogInBvbmcifX0=",
				"properties": {"priority": 0}
			}`),
			wantErr:     false,
			expectedLen: 1,
			checkField: func(response map[string]interface{}) bool {
				if workerData, exists := response["worker1@host"]; exists {
					if workerMap, ok := workerData.(map[string]interface{}); ok {
						return workerMap["ok"] == "pong"
					}
				}
				return false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := handler.ParseWorkerResponse(tt.data)

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

			if len(result) != tt.expectedLen {
				t.Errorf("Expected %d fields in response, got %d", tt.expectedLen, len(result))
			}

			if tt.checkField != nil && !tt.checkField(result) {
				t.Error("Field check failed for parsed response")
			}
		})
	}
}

func TestHandler_FormatResponse(t *testing.T) {
	handler := NewHandler()

	workerName := "worker@host"
	status := "pong"
	timestamp := time.Now()

	result := handler.FormatResponse(workerName, status, timestamp)

	if result == nil {
		t.Fatal("Expected non-nil result")
	}

	if workerData, exists := result[workerName]; !exists {
		t.Error("Expected worker data in result")
	} else {
		if workerMap, ok := workerData.(map[string]interface{}); !ok {
			t.Error("Expected worker data to be a map")
		} else {
			if ok, exists := workerMap["ok"]; !exists {
				t.Error("Expected 'ok' field in worker data")
			} else if ok != status {
				t.Errorf("Expected status %s, got %v", status, ok)
			}
		}
	}
}

func TestHandler_CreatePingMessageRaw(t *testing.T) {
	handler := NewHandler()

	tests := []struct {
		name         string
		replyTo      string
		destinations []string
		wantFields   map[string]interface{}
	}{
		{
			name:         "broadcast message raw format",
			replyTo:      "test-reply-queue",
			destinations: nil,
			wantFields: map[string]interface{}{
				"method":      "ping",
				"arguments":   map[string]interface{}{},
				"destination": nil,
				"pattern":     nil,
				"matcher":     nil,
			},
		},
		{
			name:         "targeted message raw format",
			replyTo:      "test-reply-queue",
			destinations: []string{"worker1@host1", "worker2@host2"},
			wantFields: map[string]interface{}{
				"method":      "ping",
				"arguments":   map[string]interface{}{},
				"destination": []interface{}{"worker1@host1", "worker2@host2"},
				"pattern":     nil,
				"matcher":     nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := handler.CreatePingMessage(tt.replyTo, tt.destinations, MessageFormatRaw)
			if err != nil {
				t.Fatalf("CreatePingMessage() error = %v", err)
			}

			var message map[string]interface{}
			err = json.Unmarshal(data, &message)
			if err != nil {
				t.Fatalf("Failed to parse raw ping message JSON: %v", err)
			}

			for field, expectedValue := range tt.wantFields {
				actualValue, exists := message[field]
				if !exists {
					t.Errorf("Expected field %s not found in message", field)
					continue
				}

				if expectedSlice, ok := expectedValue.([]interface{}); ok {
					actualSlice, ok := actualValue.([]interface{})
					if !ok {
						t.Errorf("Field %s: expected slice, got %T", field, actualValue)
						continue
					}
					if len(expectedSlice) != len(actualSlice) {
						t.Errorf("Field %s: expected slice length %d, got %d", field, len(expectedSlice), len(actualSlice))
						continue
					}
					for i, expected := range expectedSlice {
						if actualSlice[i] != expected {
							t.Errorf("Field %s[%d]: expected %v, got %v", field, i, expected, actualSlice[i])
						}
					}
				} else if expectedMap, ok := expectedValue.(map[string]interface{}); ok {
					actualMap, ok := actualValue.(map[string]interface{})
					if !ok {
						t.Errorf("Field %s: expected map, got %T", field, actualValue)
						continue
					}
					if len(expectedMap) != len(actualMap) {
						t.Errorf("Field %s: expected map length %d, got %d", field, len(expectedMap), len(actualMap))
						continue
					}
					if len(expectedMap) == 0 && len(actualMap) == 0 {
						// Empty maps match
						continue
					}
					t.Logf("Field %s: both maps exist with expected lengths", field)
				} else {
					if actualValue != expectedValue {
						t.Errorf("Field %s: expected %v, got %v", field, expectedValue, actualValue)
					}
				}
			}

			ticket, exists := message["ticket"]
			if !exists {
				t.Error("Expected 'ticket' field not found in message")
			} else if _, ok := ticket.(string); !ok {
				t.Errorf("Expected 'ticket' to be string, got %T", ticket)
			}

			replyToField, exists := message["reply_to"]
			if !exists {
				t.Error("Expected 'reply_to' field not found in message")
			} else {
				replyToMap, ok := replyToField.(map[string]interface{})
				if !ok {
					t.Errorf("Expected 'reply_to' to be map, got %T", replyToField)
				} else {
					if replyToMap["exchange"] != "reply.celery.pidbox" {
						t.Errorf("Expected reply_to.exchange to be 'reply.celery.pidbox', got %v", replyToMap["exchange"])
					}
					if replyToMap["routing_key"] != tt.replyTo {
						t.Errorf("Expected reply_to.routing_key to be %s, got %v", tt.replyTo, replyToMap["routing_key"])
					}
				}
			}

			// Verify the message is raw JSON (direct control message)
			t.Logf("Raw message structure verified: %s", string(data))
		})
	}
}
