package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	// Build-time variables (set via -ldflags)
	Version   = "dev"
	BuildTime = "unknown"
	GoVersion = runtime.Version()
	Platform  = fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  `Print detailed version information including build platform and commit.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("fast-celery-ping version %s\n", Version)
		fmt.Printf("Build time: %s\n", BuildTime)
		fmt.Printf("Go version: %s\n", GoVersion)
		fmt.Printf("Platform: %s\n", Platform)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

// GetVersionInfo returns formatted version information for inclusion in help text
func GetVersionInfo() string {
	return fmt.Sprintf("fast-celery-ping %s (%s)", Version, Platform)
}
