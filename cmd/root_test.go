package cmd

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"

	"fast-celery-ping/internal/broker"
	"fast-celery-ping/internal/config"

	"github.com/spf13/cobra"
)

func TestRootCommand_FlagParsing(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected func(*config.Config) bool
	}{
		{
			name: "broker URL flag",
			args: []string{"--broker-url", "redis://test:6379/1"},
			expected: func(c *config.Config) bool {
				return c.BrokerURL == "redis://test:6379/1"
			},
		},
		{
			name: "timeout flag",
			args: []string{"--timeout", "5s"},
			expected: func(c *config.Config) bool {
				return c.Timeout == 5*time.Second
			},
		},
		{
			name: "format flag",
			args: []string{"--format", "text"},
			expected: func(c *config.Config) bool {
				return c.OutputFormat == "text"
			},
		},
		{
			name: "verbose flag",
			args: []string{"--verbose"},
			expected: func(c *config.Config) bool {
				return c.Verbose == true
			},
		},
		{
			name: "database flag",
			args: []string{"--database", "2"},
			expected: func(c *config.Config) bool {
				return c.Database == 2
			},
		},
		{
			name: "username flag",
			args: []string{"--username", "testuser"},
			expected: func(c *config.Config) bool {
				return c.Username == "testuser"
			},
		},
		{
			name: "password flag",
			args: []string{"--password", "testpass"},
			expected: func(c *config.Config) bool {
				return c.Password == "testpass"
			},
		},
		{
			name: "destination flag single",
			args: []string{"--destination", "worker1@host"},
			expected: func(c *config.Config) bool {
				return len(c.Destination) == 1 && c.Destination[0] == "worker1@host"
			},
		},
		{
			name: "destination flag multiple",
			args: []string{"-d", "worker1@host,worker2@host"},
			expected: func(c *config.Config) bool {
				return len(c.Destination) == 2 && c.Destination[0] == "worker1@host" && c.Destination[1] == "worker2@host"
			},
		},
		{
			name: "destination flag with spaces",
			args: []string{"-d", "worker1@host, worker2@host, worker3@host"},
			expected: func(c *config.Config) bool {
				return len(c.Destination) == 3 &&
					c.Destination[0] == "worker1@host" &&
					c.Destination[1] == "worker2@host" &&
					c.Destination[2] == "worker3@host"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset global variables
			cfg = nil
			brokerURL = ""
			timeout = 0
			format = ""
			verbose = false
			database = 0
			username = ""
			password = ""
			destination = ""

			// Create a new root command for testing
			testCmd := &cobra.Command{
				Use: "test",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Just trigger config initialization
					return nil
				},
			}

			// Add the same flags as root command
			testCmd.PersistentFlags().StringVar(&brokerURL, "broker-url", "", "Broker URL")
			testCmd.PersistentFlags().DurationVar(&timeout, "timeout", 0, "Timeout for ping responses")
			testCmd.PersistentFlags().StringVar(&format, "format", "", "Output format")
			testCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
			testCmd.PersistentFlags().IntVar(&database, "database", 0, "Redis database number")
			testCmd.PersistentFlags().StringVar(&username, "username", "", "Redis username")
			testCmd.PersistentFlags().StringVar(&password, "password", "", "Redis password")
			testCmd.PersistentFlags().StringVarP(&destination, "destination", "d", "", "Destination node names")

			// Set OnInitialize to call our config initialization
			cobra.OnInitialize(initConfig)

			// Set args and execute
			testCmd.SetArgs(tt.args)
			err := testCmd.Execute()

			if err != nil {
				t.Fatalf("Command execution failed: %v", err)
			}

			if cfg == nil {
				t.Fatal("Config was not initialized")
			}

			if !tt.expected(cfg) {
				t.Error("Expected condition not met for parsed config")
			}
		})
	}
}

func TestOutputResults(t *testing.T) {
	tests := []struct {
		name         string
		responses    map[string]broker.PingResponse
		outputFormat string
		expectedOut  string
	}{
		{
			name: "single response JSON",
			responses: map[string]broker.PingResponse{
				"worker1@host": {
					WorkerName: "worker1@host",
					Status:     "pong",
					Timestamp:  1234567890,
				},
			},
			outputFormat: "json",
			expectedOut:  `"worker1@host": {`,
		},
		{
			name: "single response text",
			responses: map[string]broker.PingResponse{
				"worker1@host": {
					WorkerName: "worker1@host",
					Status:     "pong",
					Timestamp:  1234567890,
				},
			},
			outputFormat: "text",
			expectedOut:  "worker1@host: OK pong",
		},
		{
			name: "multiple responses JSON",
			responses: map[string]broker.PingResponse{
				"worker1@host": {
					WorkerName: "worker1@host",
					Status:     "pong",
					Timestamp:  1234567890,
				},
				"worker2@host": {
					WorkerName: "worker2@host",
					Status:     "pong",
					Timestamp:  1234567891,
				},
			},
			outputFormat: "json",
			expectedOut:  `"ok": "pong"`,
		},
		{
			name: "multiple responses text",
			responses: map[string]broker.PingResponse{
				"worker1@host": {
					WorkerName: "worker1@host",
					Status:     "pong",
					Timestamp:  1234567890,
				},
				"worker2@host": {
					WorkerName: "worker2@host",
					Status:     "pong",
					Timestamp:  1234567891,
				},
			},
			outputFormat: "text",
			expectedOut:  "2 nodes online.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Set the output format
			cfg = &config.Config{
				OutputFormat: tt.outputFormat,
			}

			// Call outputResults
			err := outputResults(tt.responses)

			// Restore stdout
			w.Close()
			os.Stdout = oldStdout

			// Read captured output
			var buf bytes.Buffer
			buf.ReadFrom(r)
			output := buf.String()

			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if !strings.Contains(output, tt.expectedOut) {
				t.Errorf("Expected output to contain '%s', got: '%s'", tt.expectedOut, output)
			}
		})
	}
}

func TestOutputResults_InvalidFormat(t *testing.T) {
	responses := map[string]broker.PingResponse{
		"worker@host": {
			WorkerName: "worker@host",
			Status:     "pong",
			Timestamp:  1234567890,
		},
	}

	cfg = &config.Config{
		OutputFormat: "invalid",
	}

	err := outputResults(responses)
	if err == nil {
		t.Error("Expected error for invalid output format")
	}

	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("Expected error about unsupported format, got: %v", err)
	}
}

func TestInitConfig_EnvVarHandling(t *testing.T) {
	// Save original environment
	originalEnv := map[string]string{
		"BROKER_URL": os.Getenv("BROKER_URL"),
	}

	defer func() {
		for key, value := range originalEnv {
			if value == "" {
				os.Unsetenv(key)
			} else {
				os.Setenv(key, value)
			}
		}
	}()

	// Test with environment variable
	os.Setenv("BROKER_URL", "redis://env-test:6379/0")

	// Reset globals
	cfg = nil
	brokerURL = ""

	// Call initConfig
	initConfig()

	if cfg == nil {
		t.Fatal("Config was not initialized")
	}

	if cfg.BrokerURL != "redis://env-test:6379/0" {
		t.Errorf("Expected broker URL from env, got: %s", cfg.BrokerURL)
	}
}

func TestInitConfig_ValidationError(t *testing.T) {
	// Save original stderr
	oldStderr := os.Stderr
	_, w, _ := os.Pipe()
	os.Stderr = w

	// Reset globals and set invalid config
	cfg = nil
	brokerURL = ""
	timeout = -time.Second // Invalid timeout

	// This should cause initConfig to call os.Exit(1)
	// We can't easily test os.Exit, but we can test that validation fails
	defer func() {
		if r := recover(); r != nil {
			// Expected - validation should fail
		}
		w.Close()
		os.Stderr = oldStderr
	}()

	// We expect this to fail validation, but we can't test os.Exit directly
	// So we'll just verify the config would be invalid
	testConfig := config.DefaultConfig()
	testConfig.Timeout = -time.Second

	err := testConfig.Validate()
	if err == nil {
		t.Error("Expected validation error for negative timeout")
	}
}
