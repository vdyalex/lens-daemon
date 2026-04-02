package cmd

import (
	"os"

	"github.com/pterm/pterm"
	"github.com/spf13/cobra"

	"github.com/vdyalex/lens-daemon/src/daemon"
	"github.com/vdyalex/lens-daemon/src/utils/buildinfo"
	"github.com/vdyalex/lens-daemon/src/utils/constants"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the daemon",
	Long:  `Stops and then starts the daemon with the specified configuration.`,
	Run: func(cmd *cobra.Command, args []string) {
		pterm.Info.Println("Restarting " + buildinfo.BinaryName + "…")
		if err := runStop(); err != nil {
			pterm.Warning.Printfln("stop skipped: %v", err)
		} else {
			if err := daemon.WaitStop(daemon.DefaultPIDPath(), constants.TimeoutDaemonStop); err != nil {
				pterm.Error.Printfln("restart aborted: %v", err)
				os.Exit(1)
			}
		}
		runStart()
	},
}
