package es

import (
	"fmt"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/fengxsong/toolkit/internal/errors"
	"github.com/fengxsong/toolkit/pkg/log"
)

type deleteOptions struct {
	*commonOptions
	namespace string
}

func newDeletePatternCommand() *cobra.Command {
	o := &deleteOptions{
		commonOptions: &commonOptions{},
	}
	cmd := &cobra.Command{
		Use:     "delete",
		Aliases: []string{"remove"},
		Short:   "Remove index pattern[s] in kibana",
		Args:    cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			o.setDefaults()
			return o.Run(args...)
		},
	}
	o.AddFlags(cmd.Flags())
	cmd.MarkFlagRequired("kibana-url")
	cmd.Flags().StringVarP(&o.namespace, "namespace", "n", "default", "Kibana namespace")

	return cmd
}

func (o *deleteOptions) Run(patterns ...string) error {
	cli, err := o.complete()
	if err != nil {
		return err
	}
	var errs []error
	for i := range patterns {
		if err = cli.deletePattern(o.namespace, patterns[i]); err != nil {
			errs = append(errs, err)
		}
		log.GetLogger().Infof("indice pattern `%s` has been removed", patterns[i])
	}
	if len(errs) == 0 {
		return nil
	}
	return errors.MultiError(errs)
}

func (c *client) deletePattern(namespace, s string) error {
	c.kibanaURL.Path = fmt.Sprintf("/s/%s/api/index_patterns/index_pattern/%s", namespace, s)
	_, err := c.doRequest(http.MethodDelete, c.kibanaURL.String(), nil, c.dryRun)
	return err
}
