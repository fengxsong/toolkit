package app

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/fengxsong/toolkit/cmd/app/factory"
	"github.com/fengxsong/toolkit/internal/version"
	"github.com/fengxsong/toolkit/pkg/log"
)

func newRootCommand() *cobra.Command {
	var dev bool
	cmd := &cobra.Command{
		Use:  "toolkit",
		Long: "Toolkit for weeget devops team",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			return log.InitLogger(dev)
		},
		SilenceErrors: true,
		SilenceUsage:  true,
	}
	cmd.PersistentFlags().BoolVar(&dev, "dev", false, "enable dev mode(logging with dev format)")
	cmd.AddCommand(factory.Registered()...)
	cmd.AddCommand(newVersionCommand())
	return cmd
}

// Run execute
func Run() error {
	root := newRootCommand()
	return root.Execute()
}

func newVersionCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "print version info",
		Run: func(_ *cobra.Command, _ []string) {
			fmt.Println(version.Version())
		},
	}
}
