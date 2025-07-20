package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	verbose      bool
	configPath   string
	outputFormat string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "taskporter",
	Short: "Cross-editor CLI task bridge - Your trusted porter for project automation",
	Long: `Taskporter - Cross-Editor CLI Task Bridge

A Death Stranding inspired CLI tool that acts as a "porter" for your development tasks.
Bridge and execute tasks from various code editors (VSCode, JetBrains IDEs) directly
from the terminal, enabling seamless cross-environment developer workflows.

Connecting isolated development environments... strand established.`,
	Version: "0.1.0",
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	setupCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// setupCommands configures all commands and their flags
func setupCommands() {
	setupGlobalFlags(rootCmd)
	setupListCommand(rootCmd)
	setupRunCommand(rootCmd)
}

// setupGlobalFlags configures the global flags for the root command
func setupGlobalFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	cmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default: auto-detect)")
	cmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format (text, json)")
}
