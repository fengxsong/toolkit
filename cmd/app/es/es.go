package es

import (
	"github.com/spf13/cobra"

	"github.com/fengxsong/toolkit/cmd/app/factory"
)

const name = "es"

func init() {
	factory.Register(name, newSubCommand())
}

func newSubCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   name,
		Short: "elasticsearch toolkit",
	}
	cmd.AddCommand(newRegsiterPatternCommand())
	return cmd
}
