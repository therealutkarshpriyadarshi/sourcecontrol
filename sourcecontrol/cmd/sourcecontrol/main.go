package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/utkarsh5026/SourceControl/pkg/common/logger"
)

var (
	Version   = "0.1.0-dev"
	BuildTime = "unknown"
	CommitSHA = "unknown"
)

var (
	logLevel  string
	logFormat string
	verbose   bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "srcc",
		Short:   "SourceControl - A Git implementation in Go",
		Long:    getBanner(),
		Version: fmt.Sprintf("%s (built: %s, commit: %s)", Version, BuildTime, CommitSHA),
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			setupLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "Log format (text, json)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output (sets log level to debug)")

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newAddCmd())
	rootCmd.AddCommand(newCommitCmd())
	rootCmd.AddCommand(newBranchCmd())
	rootCmd.AddCommand(newCheckoutCmd())
	rootCmd.AddCommand(newLogCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getBanner() string {
	return `
╔═════════════════════════════════════════════════════════════════════╗
║                                                                     ║
║   ███████╗ ██████╗ ██╗   ██╗██████╗  ██████╗███████╗                ║
║   ██╔════╝██╔═══██╗██║   ██║██╔══██╗██╔════╝██╔════╝                ║
║   ███████╗██║   ██║██║   ██║██████╔╝██║     █████╗                  ║
║   ╚════██║██║   ██║██║   ██║██╔══██╗██║     ██╔══╝                  ║
║   ███████║╚██████╔╝╚██████╔╝██║  ██║╚██████╗███████╗                ║
║   ╚══════╝ ╚═════╝  ╚═════╝ ╚═╝  ╚═╝ ╚═════╝╚══════╝                ║
║                                                                     ║
║    ██████╗ ██████╗ ███╗   ██╗████████╗██████╗  ██████╗ ██╗          ║
║   ██╔════╝██╔═══██╗████╗  ██║╚══██╔══╝██╔══██╗██╔═══██╗██║          ║
║   ██║     ██║   ██║██╔██╗ ██║   ██║   ██████╔╝██║   ██║██║          ║
║   ██║     ██║   ██║██║╚██╗██║   ██║   ██╔══██╗██║   ██║██║          ║
║   ╚██████╗╚██████╔╝██║ ╚████║   ██║   ██║  ██║╚██████╔╝███████╗     ║
║    ╚═════╝ ╚═════╝ ╚═╝  ╚═══╝   ╚═╝   ╚═╝  ╚═╝ ╚═════╝ ╚══════╝     ║
║                                                                     ║
╚═════════════════════════════════════════════════════════════════════╝

  🚀 A modern Git-like version control system implemented in Go

  📦 Version Control Made Simple
  ⚡ Fast, reliable, and easy to use
  🔧 Familiar Git-style commands
  💻 Built with Go for performance

  Get started with: srcc init
  Check status with: srcc status
  Need help? Run:   srcc --help

`
}

func setupLogging() {
	level := logger.LevelInfo
	if verbose {
		level = logger.LevelDebug
	} else {
		switch logLevel {
		case "debug":
			level = logger.LevelDebug
		case "info":
			level = logger.LevelInfo
		case "warn":
			level = logger.LevelWarn
		case "error":
			level = logger.LevelError
		}
	}

	format := logger.FormatText
	if logFormat == "json" {
		format = logger.FormatJSON
	}

	logger.Default = logger.New(logger.Config{
		Level:  level,
		Format: format,
		Output: os.Stderr,
	})
}
