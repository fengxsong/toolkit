package feishu

import (
	"github.com/spf13/cobra"

	"github.com/fengxsong/toolkit/cmd/app/factory"
)

const name = "feishu"

func init() {
	factory.Register(name, newSubCommand())
}

func newSubCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "feishu for dealing messages",
	}
	cmd.AddCommand(newSendCommand())
	cmd.AddCommand(newServeCommand())
	return cmd
}
