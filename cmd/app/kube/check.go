package kube

import (
	"fmt"
	"strings"

	"github.com/go-test/deep"
	"github.com/spf13/cobra"
	"go.uber.org/multierr"

	"github.com/fengxsong/toolkit/cmd/app/options"
)

func newCheckCommand() *cobra.Command {
	var (
		listOption = &options.KubeListOption{}
		oldFile    string
	)
	cmd := &cobra.Command{
		Use:   "check",
		Short: "Check if object(deployment) has been changed",
		RunE: func(_ *cobra.Command, _ []string) error {
			return check(listOption, oldFile)
		},
	}
	listOption.AddFlags(cmd.Flags())
	cmd.Flags().StringVarP(&oldFile, "file", "f", "", "File that contains deployment list to check")
	return cmd
}

func check(o *options.KubeListOption, filename string) error {
	cli, err := newClient()
	if err != nil {
		return err
	}
	currentItems, err := cli.listDeployments(o)
	if err != nil {
		return err
	}
	currentMap := currentItems.ToMap()

	oldItems, err := loadFromFile(filename)
	if err != nil {
		return err
	}
	oldMap := oldItems.ToMap()
	var errs []error
	for key, val := range currentMap {
		val1, ok := oldMap[key]
		if !ok {
			errs = append(errs, fmt.Errorf("%s not in history file", key))
			continue
		}
		if val1.Skip {
			continue
		}
		if diff := deep.Equal(val, val1); diff != nil {
			errs = append(errs, fmt.Errorf("%s has changed: %v", key, diff))
		}
		delete(oldMap, key)
	}
	if len(oldMap) > 0 {
		errs = append(errs, fmt.Errorf("%s has been disappeared", strings.Join(getKeys(oldMap), ", ")))
	}
	if len(errs) > 0 {
		return multierr.Combine(errs...)
	}
	return nil
}
