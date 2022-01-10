package feishu

import "github.com/spf13/cobra"

// todo: message hub
func newServeCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Serve HTTP handler for dealing messages",
	}
	return cmd
}
