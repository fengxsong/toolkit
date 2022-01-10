package kube

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/fengxsong/toolkit/cmd/app/options"
)

func newListCommand() *cobra.Command {
	var (
		listOption = &options.KubeListOption{}
		out        string
	)
	var w io.Writer
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List deployments",
		PreRunE: func(_ *cobra.Command, _ []string) error {
			if len(out) > 0 {
				fp, err := os.Create(out)
				if err != nil {
					return err
				}
				w = fp
			} else {
				w = os.Stdout
			}
			return nil
		},
		RunE: func(_ *cobra.Command, _ []string) error {
			cli, err := newClient()
			if err != nil {
				return err
			}
			items, err := cli.listDeployments(listOption)
			if err != nil {
				return err
			}
			return items.Write(w)
		},
	}
	listOption.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&out, "out", "o", "", "Write objects to file or stdout")
	return cmd
}
