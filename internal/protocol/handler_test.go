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
	}{
		{
			name:         "broadcast message",
			destinations: nil,
		},
		{
			name:         "targeted message",
			destinations: []string{"worker1@host", "worker2@host"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			messageData, err := handler.CreatePingMessage(replyTo, tt.destinations)
			if err != nil {
				t.Fatalf("Failed to create ping message: %v", err)
			}

			var envelope map[string]interface{}
			err = json.Unmarshal(messageData, &envelope)
			if err != nil {
				t.Fatalf("Failed to unmarshal ping message: %v", err)
			}

			// Check that the envelope has the expected structure
			if _, exists := envelope["body"]; !exists {
				t.Error("Expected 'body' field in ping message envelope")
			}

			if _, exists := envelope["properties"]; !exists {
				t.Error("Expected 'properties' field in ping message envelope")
			}

			if _, exists := envelope["headers"]; !exists {
				t.Error("Expected 'headers' field in ping message envelope")
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
