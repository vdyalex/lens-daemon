// Package cmd provides CLI subcommands for the daemon binary.
// The binary supports six subcommands: daemon, start, stop, status, logs, and restart.
package cmd

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/utils/buildinfo"
)

var rootCmd = &cobra.Command{
	Use:          buildinfo.BinaryName,
	Short:        "Background screenshot daemon with CLI interface",
	Long:         buildinfo.BinaryName + ` - a detached daemon that captures screenshots, extracts text, processes with Claude AI, and broadcasts to Telegram.`,
	SilenceUsage: true, // Don't print usage on errors
}

// Execute runs the root command and handles the exit code.
// It dispatches to the appropriate subcommand and exits with status 1 on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	// Register subcommands
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(logsCmd)
	rootCmd.AddCommand(restartCmd)

	// Define flags for daemon, start, restart
	for _, cmd := range []*cobra.Command{daemonCmd, startCmd, restartCmd} {
		cmd.Flags().StringVar(&flags.Model, "model", "", "Anthropic model name (env: ANTHROPIC_MODEL)")
		cmd.Flags().StringVar(&flags.SystemPrompt, "system-prompt", "", "AI system prompt (env: ANTHROPIC_SYSTEM_PROMPT)")
		cmd.Flags().IntVar(&flags.MaxTokens, "max-tokens", 0, "Max response tokens (env: ANTHROPIC_MAX_RESPONSE_TOKENS)")
		cmd.Flags().StringVar(&flags.LogLevel, "log-level", "", "Log level: debug/info/warn/error (env: LOG_LEVEL)")
		cmd.Flags().StringVar(&flags.APIKey, "api-key", "", "Anthropic API key (env: ANTHROPIC_API_KEY)")
		cmd.Flags().StringVar(&flags.BotToken, "bot-token", "", "Telegram bot token (env: TELEGRAM_BOT_TOKEN)")
		cmd.Flags().StringVar(&flags.StorePath, "store-path", "", "Subscriber store file path (env: SUBSCRIBER_STORE_PATH)")
	}
}
