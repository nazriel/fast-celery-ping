package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"fast-celery-ping/internal/broker"
	"fast-celery-ping/internal/config"

	"github.com/spf13/cobra"
)

var (
	cfg         *config.Config
	brokerURL   string
	timeout     time.Duration
	format      string
	verbose     bool
	database    int
	username    string
	password    string
	destination string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "fast-celery-ping",
	Short: "Fast alternative to celery inspect ping",
	Long: `A fast, self-contained Go alternative to 'celery inspect ping' command.
Currently supports Redis broker with easy extensibility for other brokers.

Examples:
  fast-celery-ping --broker-url redis://localhost:6379/0
  fast-celery-ping --timeout 5s --format text
  fast-celery-ping --verbose`,
	RunE: runPing,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&brokerURL, "broker-url", "", "Broker URL (default from CELERY_BROKER_URL env var or redis://localhost:6379/0)")
	rootCmd.PersistentFlags().DurationVar(&timeout, "timeout", 0, "Timeout for ping responses (default 1.5s)")
	rootCmd.PersistentFlags().StringVar(&format, "format", "", "Output format: json or text (default text)")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose output")
	rootCmd.PersistentFlags().IntVar(&database, "database", 0, "Redis database number")
	rootCmd.PersistentFlags().StringVar(&username, "username", "", "Redis username")
	rootCmd.PersistentFlags().StringVar(&password, "password", "", "Redis password")
	rootCmd.PersistentFlags().StringVarP(&destination, "destination", "d", "", "Comma separated list of destination node names")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	cfg = config.DefaultConfig()

	// Load from environment
	if err := cfg.LoadFromEnv(); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config from environment: %v\n", err)
		os.Exit(1)
	}

	// Override with command line flags
	if brokerURL != "" {
		cfg.BrokerURL = brokerURL
	}
	if timeout > 0 {
		cfg.Timeout = timeout
	}
	if format != "" {
		cfg.OutputFormat = format
	}
	if verbose {
		cfg.Verbose = verbose
	}
	if database > 0 {
		cfg.Database = database
	}
	if username != "" {
		cfg.Username = username
	}
	if password != "" {
		cfg.Password = password
	}
	if destination != "" {
		cfg.Destination = strings.Split(destination, ",")
		// Trim whitespace from each destination
		for i, dest := range cfg.Destination {
			cfg.Destination[i] = strings.TrimSpace(dest)
		}
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}
}

// runPing executes the ping command
func runPing(cmd *cobra.Command, args []string) error {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.Timeout+time.Second)
	defer cancel()

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Connecting to broker: %s\n", cfg.BrokerURL)
	}

	// Create broker
	brokerConfig := broker.Config{
		URL:      cfg.BrokerURL,
		Database: cfg.Database,
		Username: cfg.Username,
		Password: cfg.Password,
	}

	redisBroker := broker.NewRedisBroker(brokerConfig)

	// Connect to broker
	if err := redisBroker.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to broker: %w", err)
	}
	defer redisBroker.Close()

	if cfg.Verbose {
		if len(cfg.Destination) > 0 {
			fmt.Fprintf(os.Stderr, "Sending ping to specific workers: %v (timeout: %v)...\n", cfg.Destination, cfg.Timeout)
		} else {
			fmt.Fprintf(os.Stderr, "Sending ping to workers (timeout: %v)...\n", cfg.Timeout)
		}
	}

	// Execute ping
	responses, err := redisBroker.Ping(ctx, cfg.Timeout, cfg.Destination)
	if err != nil {
		return fmt.Errorf("ping failed: %w", err)
	}

	// Output results
	return outputResults(responses)
}

// outputResults formats and outputs the ping results
func outputResults(responses map[string]broker.PingResponse) error {
	if len(responses) == 0 {
		if cfg.OutputFormat == "json" {
			fmt.Println("{}")
		} else {
			fmt.Println("Error: No nodes replied within time constraint.")
		}
		os.Exit(1)
	}

	switch cfg.OutputFormat {
	case "json":
		// Format as Celery-compatible JSON
		result := make(map[string]map[string]string)
		for _, response := range responses {
			result[response.WorkerName] = map[string]string{
				"ok": response.Status,
			}
		}

		output, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON: %w", err)
		}
		fmt.Println(string(output))

	case "text":
		for _, response := range responses {
			fmt.Printf("%s: OK %s\n", response.WorkerName, response.Status)
		}
		fmt.Printf("%d nodes online.\n", len(responses))

	default:
		return fmt.Errorf("unsupported output format: %s", cfg.OutputFormat)
	}

	return nil
}
