package cmd

import (
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the daemon",
	Long:  `Stops and then starts the daemon with the specified configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.Info.Println("Restarting lensd…")
		if err := runStop(); err != nil {
			pterm.Warning.Printfln("stop skipped: %v", err)
		}
		runStart()
	},
}
