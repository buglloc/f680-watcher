package commands

import "github.com/spf13/cobra"

var rootArgs struct {
	cfgPath string
}

var rootCmd = &cobra.Command{
	Use:           "f680-watcher",
	SilenceUsage:  true,
	SilenceErrors: true,
	Short:         `f680-watcher is gold "kostyl" to fix DHCP Port Control after router reboot`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	flags := rootCmd.PersistentFlags()
	flags.StringVar(&rootArgs.cfgPath, "cfg", "", "path to config")

	rootCmd.AddCommand(
		watchCmd,
	)
}
