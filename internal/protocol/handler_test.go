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

	messageData, err := handler.CreatePingMessage(replyTo)
	if err != nil {
		t.Fatalf("Failed to create ping message: %v", err)
	}

	var message map[string]interface{}
	err = json.Unmarshal(messageData, &message)
	if err != nil {
		t.Fatalf("Failed to unmarshal ping message: %v", err)
	}

	// Check that the message has the expected structure
	if _, exists := message["data"]; !exists {
		t.Error("Expected 'data' field in ping message")
	}

	if _, exists := message["timestamp"]; !exists {
		t.Error("Expected 'timestamp' field in ping message")
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
			name: "hostname field",
			response: map[string]interface{}{
				"hostname": "worker1@host",
			},
			expected: "worker1@host",
		},
		{
			name: "worker field",
			response: map[string]interface{}{
				"worker": "worker2@host",
			},
			expected: "worker2@host",
		},
		{
			name: "nested worker info",
			response: map[string]interface{}{
				"worker": map[string]interface{}{
					"hostname": "worker3@host",
				},
			},
			expected: "worker3@host",
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
			name: "valid pong response",
			response: map[string]interface{}{
				"method": "pong",
			},
			expected: true,
		},
		{
			name: "response with hostname",
			response: map[string]interface{}{
				"hostname": "worker@host",
			},
			expected: true,
		},
		{
			name: "response with worker",
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
