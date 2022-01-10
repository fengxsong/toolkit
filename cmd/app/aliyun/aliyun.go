package aliyun

import (
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fengxsong/toolkit/cmd/app/factory"
	"github.com/fengxsong/toolkit/cmd/app/options"
)

const name = "aliyun"

func init() {
	factory.Register(name, newSubCommand(os.Stdout))
}

func newSubCommand(out io.Writer) *cobra.Command {
	o := &options.AliyunCommonOption{}
	cmd := &cobra.Command{
		Use:     name,
		Aliases: []string{"ali"},
		Short:   "aliyun toolkit",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if err := options.ExecuteRootPersistentPreRunE(cmd, args); err != nil {
				return err
			}
			return o.Validate()
		},
	}
	o.AddFlags(cmd.PersistentFlags())
	cmd.AddCommand(newSlsCommand(o, out))
	cmd.AddCommand(newOssCommand(o, out))
	return cmd
}

func visitAll(args []string, fn func(s string) error) error {
	for i := range args {
		if err := fn(args[i]); err != nil {
			return err
		}
	}
	return nil
}

func extractNames(args []string) []string {
	names := make([]string, 0)
	visitAll(args, func(s string) error {
		names = append(names, strings.Split(s, ",")...)
		return nil
	})
	return names
}
