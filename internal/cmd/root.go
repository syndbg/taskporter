package cmd

import (
	"github.com/spf13/cobra"
)

// NewRootCommand creates and configures the root command with all subcommands
func NewRootCommand() *cobra.Command {
	// Local variables for flags - no globals!
	var verbose bool
	var configPath string
	var outputFormat string

	rootCmd := &cobra.Command{
		Use:   "taskporter",
		Short: "Cross-editor CLI task bridge - Your trusted porter for project automation",
		Long: `Taskporter - Cross-Editor CLI Task Bridge

A Death Stranding inspired CLI tool that acts as a "porter" for your development tasks.
Bridge and execute tasks from various code editors (VSCode, JetBrains IDEs) directly
from the terminal, enabling seamless cross-environment developer workflows.

Connecting isolated development environments... strand established.`,
		Version: "0.1.0",
	}

	// Setup global flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "config file path (default: auto-detect)")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format (text, json)")

	_ = rootCmd.RegisterFlagCompletionFunc("output", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"text", "json"}, cobra.ShellCompDirectiveNoFileComp
	})

	rootCmd.AddCommand(NewListCommand(&verbose, &outputFormat, &configPath))
	rootCmd.AddCommand(NewRunCommand(&verbose, &configPath))
	rootCmd.AddCommand(NewPortCommand(&verbose, &configPath))

	return rootCmd
}
